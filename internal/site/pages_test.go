package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshmcarthur/design-export-docs/internal/system"
	"github.com/joshmcarthur/design-export-docs/internal/tokens"
)

// compileColours returns the colour tokens a system's tokens.json compiles to.
func compileColours(s *system.System) []tokens.ColorEntry {
	return tokens.Compile(s.Tokens, tokens.Options{}).Colors
}

// Exports contain a SKILL.md of platform boilerplate for agents. It must never become a page.
func TestExtraDocsIgnoresSkillAndReadme(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"README.md":  "# Brand book",
		"SKILL.md":   "# Platform boilerplate",
		"guide.md":   "# A guide\n\ntext",
		".hidden.md": "# hidden",
		"notes.txt":  "not markdown",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := extraDocs(dir)
	if len(got) != 1 || got[0].Stem != "guide" || got[0].Title != "A guide" {
		t.Errorf("extraDocs = %+v, want only the guide", got)
	}
}
