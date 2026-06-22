package main

import "testing"

func TestMCFamily(t *testing.T) {
	cases := map[string]string{
		"1.21.11": "1.21",
		"1.21":    "1.21",
		"1.20.4":  "1.20",
		"26.2":    "26.2", // new scheme: returned unchanged
	}
	for in, want := range cases {
		if got := mcFamily(in); got != want {
			t.Errorf("mcFamily(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReleasesNewerThan(t *testing.T) {
	m := &mojangManifest{}
	for _, v := range []struct{ id, typ string }{
		{"26.2", "release"},
		{"26.1", "release"},
		{"snap-1", "snapshot"},
		{"1.21.11", "release"},
		{"1.21.10", "release"},
	} {
		m.Versions = append(m.Versions, struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}{v.id, v.typ})
	}
	got := m.releasesNewerThan("1.21.11")
	want := []string{"26.2", "26.1"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestDiffLocksNoChange(t *testing.T) {
	l := &Lock{Minecraft: "1.21.11", Fabric: Fabric{Loader: "0.18.4", Installer: "1.1.1"}}
	if _, changed := diffLocks(l, l); changed {
		t.Error("expected no change for identical locks")
	}
}

func TestDiffLocksDetectsBump(t *testing.T) {
	old := &Lock{Minecraft: "1.21.10"}
	new := &Lock{Minecraft: "1.21.11"}
	table, changed := diffLocks(old, new)
	if !changed {
		t.Fatal("expected change")
	}
	if table == "" {
		t.Fatal("expected non-empty table")
	}
}
