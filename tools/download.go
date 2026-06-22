package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// downloadAll fetches every pinned artifact in the lock into outDir, writing
// jars to outDir/mods and datapack zips to outDir/datapacks.
func downloadAll(lock *Lock, outDir string) error {
	modsDir := filepath.Join(outDir, "mods")
	dpDir := filepath.Join(outDir, "datapacks")
	for _, d := range []string{modsDir, dpDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	// Mods: download and verify sha512.
	for _, mod := range lock.Mods {
		log.Printf("downloading mod %s (%s)", mod.Slug, mod.Version)
		data, err := download(mod.URL)
		if err != nil {
			return fmt.Errorf("download %s: %w", mod.Slug, err)
		}
		sum := sha512.Sum512(data)
		if got := hex.EncodeToString(sum[:]); got != mod.SHA512 {
			return fmt.Errorf("sha512 mismatch for %s: expected %s got %s", mod.Slug, mod.SHA512, got)
		}
		if err := os.WriteFile(filepath.Join(modsDir, mod.Filename), data, 0o644); err != nil {
			return err
		}
	}

	// Datapacks: VT bundle is a zip OF individual datapack zips -> extract each.
	if len(lock.VanillaTweaks.Datapacks) > 0 {
		if err := downloadVTBundle(vtDatapacks, lock.VanillaTweaks.Version, lock.VanillaTweaks.Datapacks, dpDir, true); err != nil {
			return err
		}
	}
	// Crafting tweaks: VT bundle IS a single combined datapack zip -> place as-is.
	if len(lock.VanillaTweaks.CraftingTweaks) > 0 {
		if err := downloadVTBundle(vtCraftingTweaks, lock.VanillaTweaks.Version, lock.VanillaTweaks.CraftingTweaks, dpDir, false); err != nil {
			return err
		}
	}
	return nil
}

// downloadVTBundle requests the bundle for the given packs and writes the result
// into dpDir. If unwrap is true the bundle contains individual datapack zips that
// are each written out; otherwise the bundle itself is a datapack and is saved as-is.
func downloadVTBundle(kind vtKind, mcFamily string, packs []LockedPack, dpDir string, unwrap bool) error {
	link, err := vtDownloadLink(kind, mcFamily, packs)
	if err != nil {
		return err
	}
	log.Printf("downloading VanillaTweaks %s bundle (%d packs)", kind.zipEndpoint, len(packs))
	data, err := download(link)
	if err != nil {
		return err
	}

	if !unwrap {
		name := fmt.Sprintf("vanillatweaks-crafting-tweaks-%s.zip", mcFamily)
		return os.WriteFile(filepath.Join(dpDir, name), data, 0o644)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("reading VT datapack bundle: %w", err)
	}
	wrote := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".zip") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		contents, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		dest := filepath.Join(dpDir, filepath.Base(f.Name))
		if err := os.WriteFile(dest, contents, 0o644); err != nil {
			return err
		}
		wrote++
	}
	if wrote == 0 {
		return fmt.Errorf("VT datapack bundle contained no .zip entries (API shape changed?)")
	}
	return nil
}
