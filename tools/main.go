// Command updater resolves and downloads the Minecraft server's mods and
// Vanilla Tweaks packs from manifest.yaml into a reproducible manifest.lock.
//
// Subcommands:
//
//	lock      resolve manifest.yaml -> manifest.lock
//	check     dry-run resolve and print the version diff (no writes)
//	download  fetch the locked artifacts into an output directory (used by the build)
//	backup    snapshot the world to Cloudflare R2 (daily, player-gated, keep last N)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "lock":
		cmdLock(os.Args[2:], false)
	case "check":
		cmdLock(os.Args[2:], true)
	case "download":
		cmdDownload(os.Args[2:])
	case "backup":
		cmdBackup(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: updater <lock|check|download|backup> [flags]")
	os.Exit(2)
}

func cmdLock(args []string, dryRun bool) {
	fs := flag.NewFlagSet("lock", flag.ExitOnError)
	root := fs.String("root", ".", "repo root containing manifest.yaml")
	upgrade := fs.Bool("upgrade", false, "also bump Minecraft (if all mods compatible) and Fabric")
	summary := fs.String("summary", "", "write a markdown change summary to this file (PR body)")
	_ = fs.Parse(args)

	manifestPath := filepath.Join(*root, "manifest.yaml")
	lockPath := filepath.Join(*root, "manifest.lock")

	m, err := loadManifest(manifestPath)
	if err != nil {
		log.Fatalf("load manifest: %v", err)
	}

	newLock, newManifest, err := resolveLock(m, *upgrade)
	if err != nil {
		log.Fatalf("resolve: %v", err)
	}

	oldLock, _ := loadLock(lockPath) // may not exist yet
	table, changed := diffLocks(oldLock, newLock)
	fmt.Println(table)

	if dryRun {
		return
	}

	if err := saveLock(lockPath, newLock); err != nil {
		log.Fatalf("write lock: %v", err)
	}
	if err := syncManifestFile(manifestPath, newManifest); err != nil {
		log.Fatalf("sync manifest: %v", err)
	}
	if *summary != "" {
		body := "## Update summary\n\n" + table + "\n"
		if err := os.WriteFile(*summary, []byte(body), 0o644); err != nil {
			log.Fatalf("write summary: %v", err)
		}
	}
	if changed {
		log.Println("lock updated")
	} else {
		log.Println("lock already current")
	}
}

func cmdDownload(args []string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	lockPath := fs.String("lock", "manifest.lock", "path to manifest.lock")
	out := fs.String("out", "out", "output directory (mods/ and datapacks/ created inside)")
	_ = fs.Parse(args)

	lock, err := loadLock(*lockPath)
	if err != nil {
		log.Fatalf("load lock: %v", err)
	}
	if err := downloadAll(lock, *out); err != nil {
		log.Fatalf("download: %v", err)
	}
	log.Println("download complete")
}
