package main

import (
	"fmt"
	"sort"
	"strings"
)

// diffLocks compares two locks and returns a markdown table of changes plus a
// bool indicating whether anything changed. old may be nil (first lock).
func diffLocks(old, new *Lock) (string, bool) {
	type change struct{ item, from, to string }
	var changes []change

	add := func(item, from, to string) {
		if from != to {
			changes = append(changes, change{item, from, to})
		}
	}

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

	add("minecraft", oldMC, new.Minecraft)
	add("fabric-loader", oldLoader, new.Fabric.Loader)
	add("fabric-installer", oldInstaller, new.Fabric.Installer)
	for _, m := range new.Mods {
		add("mod: "+m.Slug, oldMods[m.Slug], m.Version)
	}
	for _, p := range new.VanillaTweaks.Datapacks {
		add("datapack: "+p.Name, oldDP[p.Name], p.Version)
	}
	for _, p := range new.VanillaTweaks.CraftingTweaks {
		add("crafting tweak: "+p.Name, oldCT[p.Name], p.Version)
	}

	if len(changes) == 0 {
		return "No version changes.", false
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].item < changes[j].item })

	var b strings.Builder
	b.WriteString("| Item | From | To |\n|------|------|----|\n")
	for _, c := range changes {
		from := c.from
		if from == "" {
			from = "—"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | **%s** |\n", c.item, from, c.to))
	}
	return b.String(), true
}
