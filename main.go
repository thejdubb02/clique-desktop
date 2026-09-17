//go:build windows

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jchv/go-webview2"
)

func main() {
	if !claimSingleInstance() {
		return
	}

	cfg, _ := Load()

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath: WebViewDataPath(),
		WindowOptions: webview2.WindowOptions{
			Title:  "CLIque",
			Width:  1280,
			Height: 860,
		},
	})
	if w == nil {
		return
	}
	defer w.Destroy()

	if cfg.ServerURL == "" {
		_ = w.Bind("saveServer", func(rawURL, token string) string {
			u, errMsg := parseServerURL(rawURL)
			if errMsg != "" {
				return errMsg
			}
			if err := probeHealthz(u); err != nil {
				return err.Error()
			}
			cfg = Config{ServerURL: u, Token: strings.TrimSpace(token)}
			if err := Save(cfg); err != nil {
				return err.Error()
			}
			if cfg.Token != "" {
				go Watch(cfg)
			}
			w.Navigate(cfg.ServerURL)
			return ""
		})
		w.SetHtml(setupHTML)
	} else {
		w.Navigate(cfg.ServerURL)
		if cfg.Token != "" {
			go Watch(cfg)
		}
	}
	w.Run()
}

func parseServerURL(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", "enter an http or https URL"
	}
	return raw, ""
}

func probeHealthz(serverURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(strings.TrimRight(serverURL, "/") + "/healthz")
	if err != nil {
		return fmt.Errorf("cannot reach the panel: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("panel health check returned %s", res.Status)
	}
	return nil
}
