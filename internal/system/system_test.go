package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCard(t *testing.T) {
	c, ok := ParseCard(`<!-- @dsCard group="Buttons and actions" height=156 subtitle="All sizes" width=400 floor -->`)
	if !ok || c.Group != "Buttons and actions" || c.Height != 156 || c.Subtitle != "All sizes" || c.Width != 400 || !c.Floor || c.Page {
		t.Errorf("card = %+v ok=%v", c, ok)
	}
	if _, ok := ParseCard("<!doctype html>"); ok {
		t.Error("no marker should not parse")
	}
	if c, _ := ParseCard(`<!-- @dsCard page -->`); !c.Page {
		t.Error("page flag")
	}
}

func TestUnescapeName(t *testing.T) {
	if got := unescapeName("a~20b.svg"); got != "a b.svg" {
		t.Errorf("got %q", got)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsOtherFormats(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir, ""); err == nil || !strings.Contains(err.Error(), "not an exported design system") {
		t.Errorf("empty dir: %v", err)
	}
	write(t, dir, "design-system.json", `{"v":2,"layout":"files"}`)
	if _, err := Load(dir, ""); err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("v2: %v", err)
	}
}

func TestLoadMinimalSystem(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "design-system.json", `{"v":3,"layout":"files","title":"Acme","namespace":"Acme","groups":["Logos"],
	  "assetGroups":{"Logos":{"tile":"l","order":["b~20c.svg"],"files":{
	    "a.svg":{"name":"a.svg","blob":"`+strings.Repeat("a", 32)+`","size":3,"type":"image/svg+xml"},
	    "b~20c.svg":{"name":"b c.svg","blob":"`+strings.Repeat("b", 32)+`","size":4,"type":"image/svg+xml"}}}}}`)
	write(t, dir, "tokens.json", `{"name":"Acme"}`)
	write(t, dir, "components/Button/preview.html", "<!-- @dsCard group=\"Actions\" height=88 -->\n<p>x</p>")
	write(t, dir, "components/Cover/README.md", "# Cover")
	write(t, dir, "components/lib/react.js", "//")
	write(t, dir, "components/empty/.keep", "")

	s, err := Load(dir, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "Acme" || s.Slug != "acme" || s.Namespace != "Acme" {
		t.Errorf("system = %+v", s)
	}
	if len(s.Components) != 2 || s.Components[0].Name != "Button" || s.Components[0].Card.Height != 88 ||
		s.Components[0].Card.Group != "Actions" || !s.Components[1].HasReadme {
		t.Errorf("components = %+v", s.Components)
	}
	// "order" names the b~20c.svg key first; unlisted files follow, sorted.
	if len(s.Assets) != 1 || len(s.Assets[0].Files) != 2 || s.Assets[0].Files[0].Path != "assets/Logos/b c.svg" {
		t.Errorf("assets = %+v", s.Assets)
	}
	if p := s.BlobPaths()[strings.Repeat("a", 32)]; p != "assets/Logos/a.svg" {
		t.Errorf("blob path = %q", p)
	}
}

func TestSafeRel(t *testing.T) {
	for _, ok := range []string{"a", "assets/Logos/a b.svg", "fonts/x.woff2"} {
		if !SafeRel(ok) {
			t.Errorf("%q should be safe", ok)
		}
	}
	for _, bad := range []string{"", "/etc/passwd", "../x", "a/../b", "a//b", "a/./b", `a\b`, "a/", "a\x00b"} {
		if SafeRel(bad) {
			t.Errorf("%q should be unsafe", bad)
		}
	}
}

func TestLoadRejectsPathsThatEscape(t *testing.T) {
	for _, name := range []string{"../../etc/passwd", "~2e~2e/x", "/abs.svg"} {
		dir := t.TempDir()
		write(t, dir, "design-system.json", `{"v":3,"layout":"files","groups":["G"],"assetGroups":{"G":{"files":{
		  "`+name+`":{"name":"x","blob":"`+strings.Repeat("a", 32)+`","size":1,"type":"image/svg+xml"}}}}}`)
		write(t, dir, "tokens.json", `{}`)
		write(t, dir, "components/A/README.md", "x")
		if _, err := Load(dir, ""); err == nil || !strings.Contains(err.Error(), "unsafe") {
			t.Errorf("%q: err = %v", name, err)
		}
	}
	dir := t.TempDir()
	write(t, dir, "design-system.json", `{"v":3,"layout":"files","libraries":[{"name":"r","file":"../r.js"}]}`)
	write(t, dir, "tokens.json", `{}`)
	if _, err := Load(dir, ""); err == nil || !strings.Contains(err.Error(), "unsafe library") {
		t.Errorf("library: err = %v", err)
	}
}
