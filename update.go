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
	// was, and the right response is to offer no update at all rather than
	// trust an answer that does not look like our own release process.
	if !githubAsset(exeURL) || !githubAsset(sumURL) {
		exeURL, sumURL = "", ""
	}
	return tag, exeURL, sumURL, true
}

func githubAsset(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "github.com"
}

// stagedPaths is where a background download lands: the binary itself, and a
// sidecar holding the digest it was verified against. The sidecar is what
// lets a later process — the next cold start, not the one that downloaded
// it — trust the file without hitting the network again.
func stagedPaths(exe string) (newPath, sumPath string) {
	return exe + ".new", exe + ".new.sha256"
}

// stageUpdate downloads and verifies a release without touching the running
// binary, so a later restart is a rename instead of a download. Called from
// the background poll, well before anyone asks to update.
func stageUpdate(exe, exeURL, sumURL string) error {
	sumBody, err := httpGet(sumURL, 15*time.Second)
	if err != nil {
		return err
	}
	want, err := parseDigest(string(sumBody))
	if err != nil {
		return err
	}

	newPath, sumPath := stagedPaths(exe)
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
	if err := os.WriteFile(sumPath, []byte(got), 0644); err != nil {
		os.Remove(newPath)
		return err
	}
	return nil
}

// verifyStaged re-hashes a staged download against its sidecar. A fresh
// process trusts nothing it did not check itself: the file on disk could be
// a leftover from a crashed download or a release that got pulled.
func verifyStaged(exe string) bool {
	newPath, sumPath := stagedPaths(exe)
	want, err := os.ReadFile(sumPath)
	if err != nil {
		return false
	}
	f, err := os.Open(newPath)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == strings.TrimSpace(string(want))
}

// dropStaged removes a staged download and its sidecar: it either failed to
// verify or has already been swapped into place.
func dropStaged(exe string) {
	newPath, sumPath := stagedPaths(exe)
	os.Remove(newPath)
	os.Remove(sumPath)
}

// swapInStaged moves an already-verified download into the running binary's
// place. Windows will not let you overwrite a running exe, but it will let
// you rename one out of the way first.
func swapInStaged(exe string) error {
	newPath, sumPath := stagedPaths(exe)
	oldPath := exe + ".old"
	if err := os.Rename(exe, oldPath); err != nil {
		return err
	}
	if err := os.Rename(newPath, exe); err != nil {
		os.Rename(oldPath, exe)
		return err
	}
	os.Remove(sumPath)
	return nil
}

// applyStagedOrDownload is the click path: swap in whatever the background
// poll already staged, which should be instant, or download now if nothing
// is staged yet, so a click never does nothing.
func applyStagedOrDownload(exeURL, sumURL string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if !verifyStaged(exe) {
		if err := stageUpdate(exe, exeURL, sumURL); err != nil {
			return err
		}
	}
	if err := swapInStaged(exe); err != nil {
		dropStaged(exe)
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

// applyPendingUpdateIfStaged is the plain-restart path: if an earlier run
// finished staging an update before anyone closed the app, closing and
// reopening CLIque is enough on its own, no click required. Runs before the
// single-instance mutex is claimed, so there is nothing to release here.
func applyPendingUpdateIfStaged() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if !verifyStaged(exe) {
		dropStaged(exe) // a crashed download, or a release that got pulled
		return
	}
	if err := swapInStaged(exe); err != nil {
		return // keep running the old binary; the next poll tries again
	}
	if err := exec.Command(exe).Start(); err == nil {
		os.Exit(0)
	}
	// The swap already happened even though the relaunch failed. This
	// process is still the old version in memory, but the file on disk is
	// the new one, so the next restart — by any means — picks it up.
}

// cleanupOld removes the previous binary left behind by a swap.
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
