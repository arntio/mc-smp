package main

// mojangManifest is a subset of the Mojang version manifest.
type mojangManifest struct {
	Latest struct {
		Release string `json:"release"`
	} `json:"latest"`
	Versions []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"versions"`
}

// fetchMojang returns the Mojang version manifest (versions ordered newest first).
func fetchMojang() (*mojangManifest, error) {
	var m mojangManifest
	if err := getJSON("https://launchermeta.mojang.com/mc/game/version_manifest_v2.json", &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// releasesNewerThan returns release IDs published after current, newest first.
// The manifest lists versions newest -> oldest, so we collect releases until we
// reach the current version.
func (m *mojangManifest) releasesNewerThan(current string) []string {
	var out []string
	for _, v := range m.Versions {
		if v.ID == current {
			break
		}
		if v.Type == "release" {
			out = append(out, v.ID)
		}
	}
	return out
}
