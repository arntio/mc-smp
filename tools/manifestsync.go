package main

import (
	"fmt"
	"os"
	"regexp"
)

// syncManifestFile updates only the scalar version fields in manifest.yaml in
// place (minecraft, fabric.loader, fabric.installer, vanillatweaks.version),
// preserving comments, ordering and the pack lists. It is a no-op when nothing
// changed.
func syncManifestFile(path string, m *Manifest) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(data)
	for _, kv := range []struct{ key, val string }{
		{"minecraft", m.Minecraft},
		{"loader", m.Fabric.Loader},
		{"installer", m.Fabric.Installer},
		{"version", m.VanillaTweaks.Version},
	} {
		s, err = replaceScalar(s, kv.key, kv.val)
		if err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

// replaceScalar replaces the value of a `key:` line, keeping its indentation.
// The value is written quoted. The key must appear exactly once.
func replaceScalar(s, key, val string) (string, error) {
	re := regexp.MustCompile(`(?m)^(\s*` + regexp.QuoteMeta(key) + `:\s*).*$`)
	matches := re.FindAllStringIndex(s, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("manifest sync: expected exactly one %q line, found %d", key, len(matches))
	}
	return re.ReplaceAllString(s, fmt.Sprintf(`${1}"%s"`, val)), nil
}
