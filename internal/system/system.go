// Package system loads an exported design system folder.
//
// Only the layout produced by exporting a claude.ai Design System artifact is supported:
// a design-system.json index (v3, layout "files") next to tokens.json, README.md, components/,
// assets/ and fonts/.
package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/joshmcarthur/design-export-docs/internal/tokens"
)

// System is a loaded design system.
type System struct {
	Slug       string
	Dir        string
	Title      string
	Namespace  string
	Libraries  []Library
	Groups     []string // asset group order
	Assets     []AssetGroup
	LastChange *LastChange
	Tokens     *tokens.Doc
	Components []Component
}

// Library is a script every preview loads before the bundle (React, ReactDOM).
type Library struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Global  string `json:"global"`
	File    string `json:"file"`
}

// LastChange is the index's record of the most recent change.
type LastChange struct {
	By   string `json:"by"`
	At   string `json:"at"`
	Via  string `json:"via"`
	Note string `json:"note"`
}

// AssetGroup is an asset folder under assets/.
type AssetGroup struct {
	Name  string
	Tile  string
	Files []Asset // in the group's display order
}

// Asset is one uploaded file, recorded in the index by blob id.
type Asset struct {
	Name string // path below the group folder
	Blob string
	Size int64
	Type string
	Path string // path relative to the system folder, e.g. assets/Logos/logo.svg
}

// Component is a folder under components/.
type Component struct {
	Name       string
	Card       Card
	HasPreview bool
	HasReadme  bool
}

// Card is the parsed first-line "@dsCard" marker of a preview.
type Card struct {
	Present  bool
	Group    string
	Subtitle string
	Height   int
	Width    int
	Floor    bool
	Page     bool
}

type index struct {
	V          int         `json:"v"`
	Layout     string      `json:"layout"`
	Title      string      `json:"title"`
	Namespace  string      `json:"namespace"`
	Libraries  []Library   `json:"libraries"`
	Groups     []string    `json:"groups"`
	LastChange *LastChange `json:"lastChange"`
	Assets     map[string]struct {
		Tile  string   `json:"tile"`
		Order []string `json:"order"`
		Files map[string]struct {
			Name string `json:"name"`
			Blob string `json:"blob"`
			Size int64  `json:"size"`
			Type string `json:"type"`
		} `json:"files"`
	} `json:"assetGroups"`
}

// Load reads the system in dir. slug names it in URLs; empty means the folder name.
func Load(dir, slug string) (*System, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if slug == "" {
		slug = filepath.Base(abs)
	}
	raw, err := os.ReadFile(filepath.Join(abs, "design-system.json"))
	if err != nil {
		return nil, fmt.Errorf("%s: not an exported design system (no design-system.json): %w", dir, err)
	}
	var ix index
	if err := json.Unmarshal(raw, &ix); err != nil {
		return nil, fmt.Errorf("%s/design-system.json: %w", dir, err)
	}
	if ix.V != 3 || ix.Layout != "files" {
		return nil, fmt.Errorf("%s/design-system.json: unsupported format (v=%d, layout=%q); design-export-docs reads v3 \"files\" exports",
			dir, ix.V, ix.Layout)
	}
	s := &System{Slug: slug, Dir: abs, Title: ix.Title, Namespace: ix.Namespace, Libraries: ix.Libraries,
		Groups: ix.Groups, LastChange: ix.LastChange}
	if s.Title == "" {
		s.Title = slug
	}

	tj, err := os.ReadFile(filepath.Join(abs, "tokens.json"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	if s.Tokens, err = tokens.Parse(tj); err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	for _, l := range ix.Libraries {
		if !SafeRel(l.File) {
			return nil, fmt.Errorf("%s/design-system.json: unsafe library path %q", dir, l.File)
		}
	}
	if s.Assets, err = assetGroups(ix); err != nil {
		return nil, fmt.Errorf("%s/design-system.json: %w", dir, err)
	}
	if s.Components, err = components(abs); err != nil {
		return nil, err
	}
	return s, nil
}

func assetGroups(ix index) ([]AssetGroup, error) {
	names := append([]string(nil), ix.Groups...)
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	var extra []string
	for n := range ix.Assets {
		if !seen[n] {
			extra = append(extra, n)
		}
	}
	sort.Strings(extra)
	var out []AssetGroup
	for _, n := range append(names, extra...) {
		g, ok := ix.Assets[n]
		if !ok {
			continue
		}
		if !SafeRel(n) {
			return nil, fmt.Errorf("unsafe asset group name %q", n)
		}
		ag := AssetGroup{Name: n, Tile: g.Tile}
		done := map[string]bool{}
		var bad error
		add := func(key string) {
			f, ok := g.Files[key]
			if !ok || done[key] {
				return
			}
			done[key] = true
			name := unescapeName(key)
			if !SafeRel(name) {
				bad = fmt.Errorf("unsafe asset path %q in group %q", name, n)
				return
			}
			ag.Files = append(ag.Files, Asset{Name: name, Blob: f.Blob, Size: f.Size, Type: f.Type,
				Path: "assets/" + n + "/" + name})
		}
		for _, k := range g.Order {
			add(k)
		}
		var rest []string
		for k := range g.Files {
			rest = append(rest, k)
		}
		sort.Strings(rest)
		for _, k := range rest {
			add(k)
		}
		if bad != nil {
			return nil, bad
		}
		out = append(out, ag)
	}
	return out, nil
}

// SafeRel reports whether p is a plain relative path that stays inside the folder it is joined
// to: no leading slash, no backslashes, no empty, "." or ".." segments. Paths in an index come
// from the artifact's writers and are never trusted.
func SafeRel(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\\x00") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return true
}

var escRe = regexp.MustCompile(`~([0-9a-fA-F]{2})`)

// unescapeName reverses the index's "~xx" escaping of bytes outside [A-Za-z0-9_./-].
func unescapeName(k string) string {
	return escRe.ReplaceAllStringFunc(k, func(m string) string {
		var b byte
		fmt.Sscanf(m[1:], "%02x", &b)
		return string([]byte{b})
	})
}

// BlobPaths maps each asset's blob id to its path relative to the system folder.
func (s *System) BlobPaths() map[string]string {
	m := map[string]string{}
	for _, g := range s.Assets {
		for _, a := range g.Files {
			m[a.Blob] = a.Path
		}
	}
	return m
}

var (
	cardRe = regexp.MustCompile(`^\s*<!--\s*@dsCard\b(.*?)-->`)
	attrRe = regexp.MustCompile(`(\w+)(?:=(?:"([^"]*)"|(\S+)))?`)
)

// ParseCard parses a preview's first line. ok is false when there is no marker.
func ParseCard(firstLine string) (Card, bool) {
	m := cardRe.FindStringSubmatch(firstLine)
	if m == nil {
		return Card{}, false
	}
	c := Card{Present: true}
	for _, a := range attrRe.FindAllStringSubmatch(m[1], -1) {
		val := a[2]
		if val == "" {
			val = a[3]
		}
		switch a[1] {
		case "group":
			c.Group = val
		case "subtitle":
			c.Subtitle = val
		case "height":
			fmt.Sscanf(val, "%d", &c.Height)
		case "width":
			fmt.Sscanf(val, "%d", &c.Width)
		case "floor":
			c.Floor = true
		case "page":
			c.Page = true
		}
	}
	return c, true
}

func components(root string) ([]Component, error) {
	entries, err := os.ReadDir(filepath.Join(root, "components"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", root, err)
	}
	var out []Component
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "lib" {
			continue
		}
		dir := filepath.Join(root, "components", e.Name())
		c := Component{Name: e.Name()}
		if b, err := os.ReadFile(filepath.Join(dir, "preview.html")); err == nil {
			c.HasPreview = true
			line, _, _ := strings.Cut(string(b), "\n")
			c.Card, _ = ParseCard(line)
		}
		if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
			c.HasReadme = true
		}
		if c.HasPreview || c.HasReadme {
			out = append(out, c)
		}
	}
	return out, nil
}
