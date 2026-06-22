package main

import (
	"fmt"
	"log"
)

// resolveLock turns the manifest (intent) into a fully pinned Lock. It returns
// the resolved lock and a possibly-updated copy of the manifest (when --upgrade
// bumps MC or Fabric). Mods and VanillaTweaks packs always resolve to the newest
// version compatible with the resolved MC.
func resolveLock(m *Manifest, upgrade bool) (*Lock, *Manifest, error) {
	out := *m // shallow copy is fine; we only replace scalar fields/slices below

	if upgrade {
		loader, installer, err := latestStableFabric()
		if err != nil {
			return nil, nil, err
		}
		if loader != out.Fabric.Loader || installer != out.Fabric.Installer {
			log.Printf("fabric: %s/%s -> %s/%s", out.Fabric.Loader, out.Fabric.Installer, loader, installer)
		}
		out.Fabric = Fabric{Loader: loader, Installer: installer}

		target, err := pickMCBump(out.Minecraft, out.Mods)
		if err != nil {
			return nil, nil, err
		}
		if target != out.Minecraft {
			log.Printf("minecraft: %s -> %s (all mods compatible)", out.Minecraft, target)
			out.Minecraft = target
		}
	}

	// Keep the VanillaTweaks family aligned with the MC version.
	out.VanillaTweaks.Version = mcFamily(out.Minecraft)

	lock := &Lock{
		Minecraft: out.Minecraft,
		Fabric:    out.Fabric,
		VanillaTweaks: VTLock{Version: out.VanillaTweaks.Version},
	}

	for _, slug := range out.Mods {
		lm, err := resolveMod(slug, out.Minecraft)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("mod %-16s -> %s", slug, lm.Version)
		lock.Mods = append(lock.Mods, lm)
	}

	dps, err := resolveVTPacks(vtDatapacks, out.VanillaTweaks.Version, out.VanillaTweaks.Datapacks)
	if err != nil {
		return nil, nil, err
	}
	lock.VanillaTweaks.Datapacks = dps

	cts, err := resolveVTPacks(vtCraftingTweaks, out.VanillaTweaks.Version, out.VanillaTweaks.CraftingTweaks)
	if err != nil {
		return nil, nil, err
	}
	lock.VanillaTweaks.CraftingTweaks = cts

	return lock, &out, nil
}

// pickMCBump returns the newest MC release (newer than current) for which every
// mod has a fabric build, or current if none qualify (hold-until-compatible).
func pickMCBump(current string, mods []string) (string, error) {
	manifest, err := fetchMojang()
	if err != nil {
		return "", err
	}
	candidates := manifest.releasesNewerThan(current)
	for _, candidate := range candidates { // newest first
		allOK := true
		for _, slug := range mods {
			ok, err := modHasBuild(slug, candidate)
			if err != nil {
				return "", fmt.Errorf("checking %s for MC %s: %w", slug, candidate, err)
			}
			if !ok {
				allOK = false
				break
			}
		}
		if allOK {
			return candidate, nil
		}
	}
	return current, nil
}
