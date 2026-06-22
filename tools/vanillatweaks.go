package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const vtBase = "https://vanillatweaks.net"

// vtCategories mirrors the dpcategories.json / ctcategories.json structure.
type vtCategories struct {
	Categories []struct {
		Category string `json:"category"`
		Packs    []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packs"`
	} `json:"categories"`
}

// vtPackInfo is a resolved pack: its category and current version.
type vtPackInfo struct {
	Category string
	Version  string
}

// vtKind distinguishes datapacks from crafting tweaks (different endpoints).
type vtKind struct {
	categoriesFile string // e.g. "dpcategories.json"
	zipEndpoint    string // e.g. "zipdatapacks.php"
}

var (
	vtDatapacks      = vtKind{"dpcategories.json", "zipdatapacks.php"}
	vtCraftingTweaks = vtKind{"ctcategories.json", "zipcraftingtweaks.php"}
)

// fetchVTCategories returns name -> {category, version} for a VT pack kind.
func fetchVTCategories(kind vtKind, mcFamily string) (map[string]vtPackInfo, error) {
	endpoint := fmt.Sprintf("%s/assets/resources/json/%s/%s", vtBase, mcFamily, kind.categoriesFile)
	var raw vtCategories
	if err := getJSON(endpoint, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]vtPackInfo)
	for _, cat := range raw.Categories {
		for _, p := range cat.Packs {
			out[p.Name] = vtPackInfo{Category: cat.Category, Version: p.Version}
		}
	}
	return out, nil
}

// resolveVTPacks turns a list of pack names into pinned LockedPacks, failing
// loudly on any unknown name.
func resolveVTPacks(kind vtKind, mcFamily string, names []string) ([]LockedPack, error) {
	info, err := fetchVTCategories(kind, mcFamily)
	if err != nil {
		return nil, err
	}
	var packs []LockedPack
	var missing []string
	for _, name := range names {
		pi, ok := info[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		packs = append(packs, LockedPack{Name: name, Category: pi.Category, Version: pi.Version})
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("unknown VanillaTweaks %s pack(s) for MC %s: %s",
			kind.zipEndpoint, mcFamily, strings.Join(missing, ", "))
	}
	return packs, nil
}

// vtDownloadLink requests a bundle of the given packs and returns the absolute
// download URL. packs are grouped by category as the API requires.
func vtDownloadLink(kind vtKind, mcFamily string, packs []LockedPack) (string, error) {
	byCategory := map[string][]string{}
	for _, p := range packs {
		byCategory[p.Category] = append(byCategory[p.Category], p.Name)
	}
	packsJSON, err := json.Marshal(byCategory)
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("version", mcFamily)
	form.Set("packs", string(packsJSON))

	var resp struct {
		Status string `json:"status"`
		Link   string `json:"link"`
		Message string `json:"message"`
	}
	endpoint := fmt.Sprintf("%s/assets/server/%s", vtBase, kind.zipEndpoint)
	if err := postForm(endpoint, form, &resp); err != nil {
		return "", err
	}
	if resp.Status != "success" || resp.Link == "" {
		return "", fmt.Errorf("VanillaTweaks %s failed: status=%q message=%q", kind.zipEndpoint, resp.Status, resp.Message)
	}
	return vtBase + resp.Link, nil
}
