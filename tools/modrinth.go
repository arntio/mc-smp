package main

import (
	"fmt"
	"net/url"
	"sort"
)

// modrinthVersion is a subset of the Modrinth v2 version object.
type modrinthVersion struct {
	VersionNumber string `json:"version_number"`
	DatePublished string `json:"date_published"`
	Files         []struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Primary  bool   `json:"primary"`
		Hashes   struct {
			SHA512 string `json:"sha512"`
		} `json:"hashes"`
	} `json:"files"`
}

// modrinthVersions lists fabric versions of a project compatible with the given
// MC version, newest (by publish date) first.
func modrinthVersions(slug, mcVersion string) ([]modrinthVersion, error) {
	q := url.Values{}
	q.Set("loaders", `["fabric"]`)
	q.Set("game_versions", fmt.Sprintf(`["%s"]`, mcVersion))
	endpoint := "https://api.modrinth.com/v2/project/" + url.PathEscape(slug) + "/version?" + q.Encode()

	var versions []modrinthVersion
	if err := getJSON(endpoint, &versions); err != nil {
		return nil, err
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].DatePublished > versions[j].DatePublished
	})
	return versions, nil
}

// resolveMod returns the newest compatible Modrinth file for a mod, pinned.
func resolveMod(slug, mcVersion string) (LockedMod, error) {
	versions, err := modrinthVersions(slug, mcVersion)
	if err != nil {
		return LockedMod{}, err
	}
	if len(versions) == 0 {
		return LockedMod{}, fmt.Errorf("mod %q has no fabric build for MC %s", slug, mcVersion)
	}
	v := versions[0]
	file := v.Files[0]
	for _, f := range v.Files {
		if f.Primary {
			file = f
			break
		}
	}
	if file.Hashes.SHA512 == "" || file.URL == "" {
		return LockedMod{}, fmt.Errorf("mod %q version %s missing url/sha512", slug, v.VersionNumber)
	}
	return LockedMod{
		Slug:     slug,
		Version:  v.VersionNumber,
		Filename: file.Filename,
		URL:      file.URL,
		SHA512:   file.Hashes.SHA512,
	}, nil
}

// modHasBuild reports whether a mod has any fabric build for the given MC version.
func modHasBuild(slug, mcVersion string) (bool, error) {
	versions, err := modrinthVersions(slug, mcVersion)
	if err != nil {
		return false, err
	}
	return len(versions) > 0, nil
}
