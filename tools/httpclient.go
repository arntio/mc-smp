package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// userAgent identifies this tool to upstream APIs (Modrinth asks for one).
const userAgent = "arntio/mc-smp updater (+https://github.com/arntio/mc-smp)"

var httpClient = &http.Client{Timeout: 60 * time.Second}

// getJSON fetches url and decodes the JSON body into out.
func getJSON(rawurl string, out any) error {
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: status %d: %s", rawurl, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// postForm posts form values and decodes the JSON body into out.
func postForm(rawurl string, form url.Values, out any) error {
	req, err := http.NewRequest(http.MethodPost, rawurl, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("POST %s: status %d: %s", rawurl, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// download fetches rawurl and returns the raw bytes.
func download(rawurl string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", rawurl, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
