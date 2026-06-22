package main

// Manifest is the human-edited source of intent (manifest.yaml).
type Manifest struct {
	Minecraft     string   `yaml:"minecraft"`
	Fabric        Fabric   `yaml:"fabric"`
	Mods          []string `yaml:"mods"`
	VanillaTweaks VTIntent `yaml:"vanillatweaks"`
}

// Fabric pins the loader and installer (launcher) versions.
type Fabric struct {
	Loader    string `yaml:"loader"`
	Installer string `yaml:"installer"`
}

// VTIntent lists the Vanilla Tweaks packs we want, by display name.
type VTIntent struct {
	Version        string   `yaml:"version"` // MC family used by the VT API, e.g. "1.21"
	Datapacks      []string `yaml:"datapacks"`
	CraftingTweaks []string `yaml:"craftingtweaks"`
}

// Lock is the generated, reproducible resolution (manifest.lock).
type Lock struct {
	Minecraft     string      `yaml:"minecraft"`
	Fabric        Fabric      `yaml:"fabric"`
	Mods          []LockedMod `yaml:"mods"`
	VanillaTweaks VTLock      `yaml:"vanillatweaks"`
}

// LockedMod is a fully pinned, integrity-checked Modrinth file.
type LockedMod struct {
	Slug     string `yaml:"slug"`
	Version  string `yaml:"version"` // Modrinth version_number
	Filename string `yaml:"filename"`
	URL      string `yaml:"url"`    // stable Modrinth CDN URL
	SHA512   string `yaml:"sha512"` // verified on download
}

// VTLock pins each pack's name, category (needed by the download API) and version.
// VT download links are ephemeral, so they are re-requested at build time.
type VTLock struct {
	Version        string       `yaml:"version"`
	Datapacks      []LockedPack `yaml:"datapacks"`
	CraftingTweaks []LockedPack `yaml:"craftingtweaks"`
}

// LockedPack is one Vanilla Tweaks pack.
type LockedPack struct {
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
	Version  string `yaml:"version"`
}
