package main

import "fmt"

type fabricEntry struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// latestStableFabric returns the newest stable loader and installer versions.
func latestStableFabric() (loader, installer string, err error) {
	loader, err = firstStable("https://meta.fabricmc.net/v2/versions/loader")
	if err != nil {
		return "", "", fmt.Errorf("fabric loader: %w", err)
	}
	installer, err = firstStable("https://meta.fabricmc.net/v2/versions/installer")
	if err != nil {
		return "", "", fmt.Errorf("fabric installer: %w", err)
	}
	return loader, installer, nil
}

func firstStable(endpoint string) (string, error) {
	var entries []fabricEntry
	if err := getJSON(endpoint, &entries); err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Stable {
			return e.Version, nil
		}
	}
	if len(entries) > 0 {
		return entries[0].Version, nil
	}
	return "", fmt.Errorf("no versions returned from %s", endpoint)
}
