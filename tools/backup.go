package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type backupConfig struct {
	rconAddr     string
	rconPassword string
	dataDir      string
	paths        []string
	interval     time.Duration
	playerPoll   time.Duration
	keep         int
	prefix       string
	r2           *R2
}

const stateSuffix = "/.last-fingerprint"

func loadBackupConfig() (*backupConfig, error) {
	host := env("RCON_HOST", "localhost")
	port := env("RCON_PORT", "25575")
	pass := os.Getenv("RCON_PASSWORD")
	if pass == "" {
		return nil, fmt.Errorf("RCON_PASSWORD is required")
	}

	interval, err := time.ParseDuration(env("BACKUP_INTERVAL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("BACKUP_INTERVAL: %w", err)
	}
	poll, err := time.ParseDuration(env("BACKUP_PLAYER_POLL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("BACKUP_PLAYER_POLL: %w", err)
	}
	keep, err := strconv.Atoi(env("BACKUP_KEEP", "7"))
	if err != nil || keep < 1 {
		return nil, fmt.Errorf("BACKUP_KEEP must be a positive integer")
	}

	endpoint := os.Getenv("R2_ENDPOINT")
	if endpoint == "" {
		acct := os.Getenv("R2_ACCOUNT_ID")
		if acct == "" {
			return nil, fmt.Errorf("set R2_ENDPOINT or R2_ACCOUNT_ID")
		}
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", acct)
	}
	bucket := os.Getenv("R2_BUCKET")
	ak := os.Getenv("R2_ACCESS_KEY_ID")
	sk := os.Getenv("R2_SECRET_ACCESS_KEY")
	if bucket == "" || ak == "" || sk == "" {
		return nil, fmt.Errorf("R2_BUCKET, R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY are required")
	}

	paths := splitCSV(env("BACKUP_PATHS", "world,whitelist.json,ops.json,banned-players.json,banned-ips.json,usercache.json"))

	return &backupConfig{
		rconAddr:     net.JoinHostPort(host, port),
		rconPassword: pass,
		dataDir:      env("BACKUP_DATA_DIR", "/data"),
		paths:        paths,
		interval:     interval,
		playerPoll:   poll,
		keep:         keep,
		prefix:       env("BACKUP_PREFIX", "snake-smp"),
		r2:           newR2(endpoint, bucket, ak, sk),
	}, nil
}

func cmdBackup(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	once := fs.Bool("once", false, "run a single attempt and exit")
	force := fs.Bool("force", false, "bypass the player/no-change checks")
	dryRun := fs.Bool("dry-run", false, "decide and log but do not save-off/upload/prune")
	_ = fs.Parse(args)

	cfg, err := loadBackupConfig()
	if err != nil {
		log.Fatalf("backup config: %v", err)
	}
	ctx := context.Background()

	if *once {
		seen := true
		if !*force {
			n, err := rconPlayerCount(cfg.rconAddr, cfg.rconPassword)
			seen = err == nil && n > 0
		}
		if err := attempt(ctx, cfg, seen, *force, *dryRun); err != nil {
			log.Fatalf("backup: %v", err)
		}
		return
	}

	runLoop(ctx, cfg, *dryRun)
}

// runLoop backs up every interval, but only when at least one player has been
// seen online since the last backup (idle periods are skipped) and the world
// actually changed.
func runLoop(ctx context.Context, cfg *backupConfig, dryRun bool) {
	log.Printf("backup loop: every %s, keep last %d, repo %s/%s", cfg.interval, cfg.keep, cfg.r2.bucket, cfg.prefix)
	sawPlayers := false
	next := time.Now().Add(cfg.interval)
	for {
		// Wait for the next tick while polling for player activity.
		for time.Now().Before(next) {
			wait := cfg.playerPoll
			if d := time.Until(next); d < wait {
				wait = d
			}
			time.Sleep(wait)
			if n, err := rconPlayerCount(cfg.rconAddr, cfg.rconPassword); err != nil {
				log.Printf("player poll: %v", err)
			} else if n > 0 {
				sawPlayers = true
			}
		}
		next = next.Add(cfg.interval)

		if err := attempt(ctx, cfg, sawPlayers, false, dryRun); err != nil {
			log.Printf("backup failed (will retry next cycle): %v", err)
			continue // keep sawPlayers so we retry
		}
		sawPlayers = false
	}
}

// attempt runs one backup decision: skip if no players seen, skip if unchanged,
// otherwise snapshot the world to R2 and prune to the last N.
func attempt(ctx context.Context, cfg *backupConfig, sawPlayers, force, dryRun bool) error {
	if !sawPlayers && !force {
		log.Printf("skip: no players seen since last backup")
		return nil
	}

	roots := resolvePaths(cfg.dataDir, cfg.paths)
	if len(roots) == 0 {
		return fmt.Errorf("none of the backup paths exist under %s", cfg.dataDir)
	}

	fp, err := fingerprint(cfg.dataDir, roots)
	if err != nil {
		return fmt.Errorf("fingerprint: %w", err)
	}
	stateKey := cfg.prefix + stateSuffix
	last, _, err := cfg.r2.getText(ctx, stateKey)
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	if fp == last && !force {
		log.Printf("skip: world unchanged since last backup")
		return nil
	}

	key := fmt.Sprintf("%s/%s-%s.tar.gz", cfg.prefix, cfg.prefix, time.Now().UTC().Format("20060102T150405Z"))
	if dryRun {
		log.Printf("[dry-run] would back up %d path(s) -> %s", len(roots), key)
		return nil
	}

	// Flush the world and stop writes for a consistent snapshot.
	if err := cfg.rconSave(false); err != nil {
		return fmt.Errorf("save-off/flush: %w", err)
	}
	defer func() {
		if err := cfg.rconSave(true); err != nil {
			log.Printf("WARNING: failed to re-enable saving (save-on): %v", err)
		}
	}()

	log.Printf("uploading %s", key)
	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(writeTarGz(pw, cfg.dataDir, roots))
	}()
	if err := cfg.r2.upload(ctx, key, pr); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	if err := cfg.r2.putText(ctx, stateKey, fp); err != nil {
		log.Printf("WARNING: failed to record fingerprint: %v", err)
	}
	if err := cfg.prune(ctx); err != nil {
		log.Printf("WARNING: prune failed: %v", err)
	}
	log.Printf("backup complete: %s", key)
	return nil
}

// rconSave toggles world saving and flushes on save-off.
func (cfg *backupConfig) rconSave(on bool) error {
	rc, err := rconDial(cfg.rconAddr, cfg.rconPassword)
	if err != nil {
		return err
	}
	defer rc.Close()
	if on {
		_, err = rc.exec("save-on")
		return err
	}
	if _, err = rc.exec("save-off"); err != nil {
		return err
	}
	_, err = rc.exec("save-all flush")
	if err != nil {
		return err
	}
	time.Sleep(3 * time.Second) // give the flush a moment to settle
	return nil
}

// prune deletes the oldest backups beyond the keep count.
func (cfg *backupConfig) prune(ctx context.Context) error {
	keys, err := cfg.r2.listKeys(ctx, cfg.prefix+"/"+cfg.prefix+"-")
	if err != nil {
		return err
	}
	var backups []string
	for _, k := range keys {
		if strings.HasSuffix(k, ".tar.gz") {
			backups = append(backups, k)
		}
	}
	if len(backups) <= cfg.keep {
		return nil
	}
	for _, k := range backups[:len(backups)-cfg.keep] { // sorted ascending => oldest first
		log.Printf("pruning old backup %s", k)
		if err := cfg.r2.delete(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
