package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	_ "embed"
)

//go:embed update.js
var updateJS string

const latestURL = "https://api.github.com/repos/thejdubb02/clique-desktop/releases/latest"
const exeAsset = "CLIque.exe"
const sumAsset = "CLIque.exe.sha256"

// The package name and the application id inside it, both "Clique". Windows
// needs the first to find the installed package and the second to start it.
const packageName = "Clique"
const packageAppID = "Clique"

// isNewer compares dotted versions a segment at a time, splitting on '.' and
// '-'. A segment that is not a number counts as 0 rather than panicking: a
// tag published by hand must never be able to crash the app.
func isNewer(candidate, running string) bool {
	a := versionParts(candidate)
	b := versionParts(running)
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		left, right := 0, 0
		if i < len(a) {
			left = a[i]
		}
		if i < len(b) {
			right = b[i]
		}
		if left != right {
			return left > right
		}
	}
	return false
}

func versionParts(s string) []int {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == '-'
	})
	out := make([]int, len(fields))
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			continue
		}
		out[i] = n
	}
	return out
}

// newerRelease returns a published release newer than current. Every failure
// path returns ok=false and no error: an update check that failed is not
// something to interrupt someone's work about.
func newerRelease(current string) (version, exeURL, sumURL string, ok bool) {
	if current == "dev" {
		return
	}
	req, err := http.NewRequest(http.MethodGet, latestURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return
	}
	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return
	}
	tag := strings.TrimPrefix(payload.TagName, "v")
	if !isNewer(tag, current) {
		return
	}
	for _, a := range payload.Assets {
		switch a.Name {
		case exeAsset:
			exeURL = a.URL
		case sumAsset:
			sumURL = a.URL
		}
	}
	// This app downloads something and then executes it, so where it downloads
	// from is a trust boundary and not a detail. Asset URLs always live on
	// github.com; anything else means the API answer was not what we think it
	// was, and the right response is to offer no loose binary at all.
	//
	// Missing or untrusted assets are emptied rather than failing the whole
	// check, because a packaged install updates from the manifest and never
	// touches them. Failing here would have meant a release that dropped the
	// loose exe silently stopped offering updates to everybody.
	if !githubAsset(exeURL) || !githubAsset(sumURL) {
		exeURL, sumURL = "", ""
	}
	return tag, exeURL, sumURL, true
}

// packagedRestartCommand is what starts the app again after this copy quits.
//
// It no longer installs anything. Replacing the package from inside the app
// being replaced failed four different ways on 2026-09-18, every one of them
// ending with the app shut down and nothing running, and the last two were
// invisible from the Linux box this is built on. Windows already keeps an MSIX
// current through the manifest's background task, so the button's job is the
// part Windows does not do: get you onto the version that is installed.
//
// If Windows has not staged the new one yet, restarting lands on the same
// version and the card comes back. That is a wasted click. It is not somebody
// left with no application.
//
// The family name carries a hash of the signing identity, so it is asked for
// rather than written down, and the launch is guarded on having one. The delay
// is for this process to finish exiting: the single instance check would
// otherwise find the old copy still holding the mutex.
func packagedRestartCommand() []string {
	return []string{
		"powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command",
		"$n = '" + packageName + "'; " +
			"$f = (Get-AppxPackage -Name $n | Select-Object -First 1).PackageFamilyName; " +
			"Start-Sleep -Milliseconds 2500; " +
			"if ($f) { Start-Process ('shell:appsFolder\\' + $f + '!" + packageAppID + "') }",
	}
}

// restartPackaged queues the relaunch and hands back. The caller quits.
func restartPackaged() error {
	releaseSingleInstance()
	if err := startDetached(packagedRestartCommand()); err != nil {
		claimSingleInstance()
		return err
	}
	return nil
}

func githubAsset(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "github.com"
}

func applyUpdate(exeURL, sumURL string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	sumBody, err := httpGet(sumURL, 15*time.Second)
	if err != nil {
		return err
	}
	want, err := parseDigest(string(sumBody))
	if err != nil {
		return err
	}

	newPath := exe + ".new"
	got, err := downloadHashed(exeURL, newPath)
	if err != nil {
		os.Remove(newPath)
		return err
	}
	// A digest mismatch means the download was corrupt or truncated.
	// It is not a defence against a compromised release; that is what
	// code signing would be for.
	if got != want {
		os.Remove(newPath)
		return fmt.Errorf("download did not match its checksum")
	}

	oldPath := exe + ".old"
	if err := os.Rename(exe, oldPath); err != nil {
		os.Remove(newPath)
		return err
	}
	if err := os.Rename(newPath, exe); err != nil {
		os.Rename(oldPath, exe)
		os.Remove(newPath)
		return err
	}

	// The named mutex is still held by this process. Close it before
	// starting the new one, or the new process will see us and exit.
	releaseSingleInstance()
	if err := exec.Command(exe).Start(); err != nil {
		// The swap happened but nothing was launched, so this process carries
		// on as the only copy. Take the mutex back, or a second launch would
		// open a second window.
		claimSingleInstance()
		return err
	}
	os.Exit(0)
	return nil
}

// cleanupOld removes the previous binary left behind by applyUpdate.
// It cannot be done at update time: the old binary is still running then.
func cleanupOld() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = os.Remove(exe + ".old")
}

func parseDigest(s string) (string, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", fmt.Errorf("checksum file is empty")
	}
	sum := strings.ToLower(fields[0])
	b, err := hex.DecodeString(sum)
	if err != nil || len(b) != 32 {
		return "", fmt.Errorf("checksum is not 64 hex characters")
	}
	return sum, nil
}

func httpGet(rawURL string, timeout time.Duration) ([]byte, error) {
	res, err := (&http.Client{Timeout: timeout}).Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %s", res.Status)
	}
	return io.ReadAll(res.Body)
}

func downloadHashed(rawURL, path string) (string, error) {
	res, err := (&http.Client{Timeout: 2 * time.Minute}).Get(rawURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %s", res.Status)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, h), res.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
