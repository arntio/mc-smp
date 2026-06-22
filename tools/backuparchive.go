package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// resolvePaths returns the existing absolute paths to include in a backup,
// skipping any that don't exist (e.g. banned-ips.json before first use).
func resolvePaths(dataDir string, rel []string) []string {
	var out []string
	for _, r := range rel {
		p := filepath.Join(dataDir, r)
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// fingerprint hashes the set of (relative path, size, modtime) for every file
// under the given roots. Identical content+metadata => identical fingerprint,
// so an unchanged world is detected and skipped.
func fingerprint(dataDir string, roots []string) (string, error) {
	var lines []string
	for _, root := range roots {
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(dataDir, p)
			lines = append(lines, fmt.Sprintf("%s\t%d\t%d", rel, info.Size(), info.ModTime().UnixNano()))
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(lines)
	h := sha256.New()
	for _, l := range lines {
		io.WriteString(h, l)
		io.WriteString(h, "\n")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeTarGz streams a gzip-compressed tar of roots (paths stored relative to
// dataDir) into w.
func writeTarGz(w io.Writer, dataDir string, roots []string) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	for _, root := range roots {
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() && !info.IsDir() {
				return nil // skip symlinks/sockets
			}
			rel, err := filepath.Rel(dataDir, p)
			if err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = filepath.ToSlash(rel)
			if info.IsDir() {
				hdr.Name += "/"
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		})
		if err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}
