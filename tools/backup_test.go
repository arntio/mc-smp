package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPlayerCountRe(t *testing.T) {
	cases := map[string]string{
		"There are 0 of a max of 20 players online: ":        "0",
		"There are 3 of a max of 20 players online: a, b, c": "3",
	}
	for in, want := range cases {
		m := playerCountRe.FindStringSubmatch(in)
		if m == nil || m[1] != want {
			t.Errorf("parse %q => %v, want %s", in, m, want)
		}
	}
}

func TestFingerprintChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	world := filepath.Join(dir, "world")
	if err := os.MkdirAll(world, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(world, "level.dat")
	if err := os.WriteFile(f, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	fp1, err := fingerprint(dir, []string{world})
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := fingerprint(dir, []string{world})
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Fatal("fingerprint should be stable for identical trees")
	}

	// Changing size must change the fingerprint.
	if err := os.WriteFile(f, []byte("abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp3, err := fingerprint(dir, []string{world})
	if err != nil {
		t.Fatal(err)
	}
	if fp3 == fp1 {
		t.Fatal("fingerprint should change when a file changes")
	}
}

func TestWriteTarGzRoundTrip(t *testing.T) {
	dir := t.TempDir()
	world := filepath.Join(dir, "world")
	if err := os.MkdirAll(world, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(world, "level.dat"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := writeTarGz(&buf, dir, []string{world}); err != nil {
		t.Fatalf("writeTarGz: %v", err)
	}

	gz, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == "world/level.dat" {
			b, _ := io.ReadAll(tr)
			if string(b) != "hello" {
				t.Fatalf("content = %q, want hello", b)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("world/level.dat not found in archive")
	}
}

func TestResolvePathsSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "whitelist.json"), []byte("[]"), 0o644)
	got := resolvePaths(dir, []string{"world", "whitelist.json", "ops.json"})
	if len(got) != 1 || filepath.Base(got[0]) != "whitelist.json" {
		t.Fatalf("expected only whitelist.json, got %v", got)
	}
}
