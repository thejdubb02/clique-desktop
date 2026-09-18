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

// The manifest Windows reads to find the current package. It always names the
// newest one, which is why the packaged path needs no asset URL of its own.
const appInstallerURL = "https://github.com/thejdubb02/clique-desktop/releases/latest/download/clique.appinstaller"

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

// packagedUpdateCommand is what replaces an MSIX install with the published
// version and starts it again.
//
// Windows will not replace a package while it is running, which is what
// ForceTargetApplicationShutdown is for. If the install fails the app is
// started again anyway rather than leaving somebody with nothing running, and
// because the version will not have changed, the next check offers the update
// again. A failure that corrects itself beats a marker file nobody reads.
//
// The package family name is asked for rather than written down: it carries a
// hash of the signing identity, so a hardcoded one would quietly stop matching
// the day that key is replaced.
func packagedUpdateCommand() []string {
	return []string{
		"powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command",
		"$n = '" + packageName + "'; " +
			// Read the family name before the install, not after. Get-AppxPackage
			// can come back empty in the moment a package is being replaced, and
			// an empty name built a launch path that goes nowhere, which left the
			// app shut down by ForceTargetApplicationShutdown and nothing started
			// in its place.
			"$f = (Get-AppxPackage -Name $n | Select-Object -First 1).PackageFamilyName; " +
			"try { Add-AppxPackage -AppInstallerFile '" + appInstallerURL + "' -ForceTargetApplicationShutdown } " +
			"catch { }; " +
			"if (-not $f) { $f = (Get-AppxPackage -Name $n | Select-Object -First 1).PackageFamilyName }; " +
			"if ($f) { Start-Process ('shell:appsFolder\\' + $f + '!" + packageAppID + "') }",
	}
}

// applyPackagedUpdate pulls the published package in and restarts into it.
//
// It deliberately does not go through the package's own launcher. That launcher
// only checks for an update when Conveyor is set to aggressive, which we are
// not, because aggressive makes every cold start wait on the network before the
// window appears. Run any other way it just launches the app, so the button
// would have restarted without updating.
func applyPackagedUpdate() error {
	cmd := packagedUpdateCommand()
	// Add-AppxPackage shuts this process down itself, so the mutex would be
	// released by the operating system anyway. Doing it first is cheap and
	// removes the window where a restart races our own teardown.
	releaseSingleInstance()
	if err := startDetached(cmd); err != nil {
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
