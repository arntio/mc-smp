package main

import (
	"fmt"
	"sort"
	"strings"
)

// diffLocks compares two locks and returns a markdown changelog plus a bool
// indicating whether anything changed. Core platform versions render as plain
// lines; mods, datapacks and crafting tweaks collapse into <details> blocks.
// old may be nil (first lock).
func diffLocks(old, new *Lock) (string, bool) {
	var oldMC, oldLoader, oldInstaller string
	oldMods := map[string]string{}
	oldDP := map[string]string{}
	oldCT := map[string]string{}
	if old != nil {
		oldMC, oldLoader, oldInstaller = old.Minecraft, old.Fabric.Loader, old.Fabric.Installer
		for _, m := range old.Mods {
			oldMods[m.Slug] = m.Version
		}
		for _, p := range old.VanillaTweaks.Datapacks {
			oldDP[p.Name] = p.Version
		}
		for _, p := range old.VanillaTweaks.CraftingTweaks {
			oldCT[p.Name] = p.Version
		}
	}

	dash := func(s string) string {
		if s == "" {
			return "—"
		}
		return s
	}
	changed := false

	// Core platform versions render as plain lines at the top.
	var head []string
	core := func(label, from, to string) {
		if from != to {
			changed = true
			head = append(head, fmt.Sprintf("**%s:** %s → **%s**", label, dash(from), to))
		}
	}
	core("Minecraft", oldMC, new.Minecraft)
	core("Fabric loader", oldLoader, new.Fabric.Loader)
	core("Fabric installer", oldInstaller, new.Fabric.Installer)

	// row formats a single changed entry, or "" when unchanged.
	row := func(name, from, to string) string {
		if from == to {
			return ""
		}
		return fmt.Sprintf("- %s: %s → **%s**", name, dash(from), to)
	}
	// section wraps changed rows in a collapsible block, or "" when empty.
	section := func(title string, rows []string) string {
		var kept []string
		for _, r := range rows {
			if r != "" {
				kept = append(kept, r)
			}
		}
		if len(kept) == 0 {
			return ""
		}
		changed = true
		sort.Strings(kept)
		return fmt.Sprintf("<details>\n<summary>%s (%d)</summary>\n\n%s\n\n</details>",
			title, len(kept), strings.Join(kept, "\n"))
	}

	var modRows, dpRows, ctRows []string
	for _, m := range new.Mods {
		modRows = append(modRows, row(m.Slug, oldMods[m.Slug], m.Version))
	}
	for _, p := range new.VanillaTweaks.Datapacks {
		dpRows = append(dpRows, row(p.Name, oldDP[p.Name], p.Version))
	}
	for _, p := range new.VanillaTweaks.CraftingTweaks {
		ctRows = append(ctRows, row(p.Name, oldCT[p.Name], p.Version))
	}

	var blocks []string
	if len(head) > 0 {
		blocks = append(blocks, strings.Join(head, "\n"))
	}
	for _, s := range []string{
		section("Mods", modRows),
		section("Datapacks", dpRows),
		section("Crafting tweaks", ctRows),
	} {
		if s != "" {
			blocks = append(blocks, s)
		}
	}

	if !changed {
		return "No version changes.", false
	}
	return strings.Join(blocks, "\n\n") + "\n", true
}
