package site

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joshmcarthur/design-export-docs/internal/fixtures"
	"github.com/joshmcarthur/design-export-docs/internal/system"
)

// buildFixtures builds every fixture (the synthetic export, plus any in DED_FIXTURES) into one site.
func buildFixtures(t *testing.T) (out string, systems []*system.System, mini *system.System) {
	t.Helper()
	for _, f := range fixtures.All(t) {
		s, err := system.Load(f.Dir, f.Slug)
		if err != nil {
			t.Fatal(err)
		}
		systems = append(systems, s)
		if f.Mini {
			mini = s
		}
	}
	out = filepath.Join(t.TempDir(), "site")
	res, err := Build(systems, out)
	if err != nil {
		t.Fatal(err)
	}
	// The synthetic export has one invalid colour on purpose; nothing else may warn.
	for _, w := range res.Warnings {
		if !strings.Contains(w, "broken") {
			t.Errorf("unexpected warning: %s", w)
		}
	}
	return out, systems, mini
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func textual(p string) bool {
	switch filepath.Ext(p) {
	case ".html", ".css", ".js", ".json":
		return true
	}
	return false
}

func TestBuildFixtures(t *testing.T) {
	out, systems, _ := buildFixtures(t)

	for _, s := range systems {
		root := filepath.Join(out, s.Slug)
		for _, f := range []string{"index.html", "tokens.css", "tokens.json", "components/index.html", "components/bundle.js"} {
			if !exists(filepath.Join(root, f)) {
				t.Errorf("%s: missing %s", s.Slug, f)
			}
		}
		// Every component has a page, and a wrapped preview that loads what it needs.
		for _, c := range s.Components {
			if c.Name != "Cover" && !exists(filepath.Join(root, "components", c.Name, "index.html")) {
				t.Errorf("%s: no page for %s", s.Slug, c.Name)
			}
			if !c.HasPreview {
				continue
			}
			pv := read(t, filepath.Join(root, "components", c.Name, "preview.html"))
			for _, want := range []string{`href="../../tokens.css"`, `src="../bundle.js"`, `data-theme="light"`} {
				if !strings.Contains(pv, want) {
					t.Errorf("%s/%s preview lacks %s", s.Slug, c.Name, want)
				}
			}
			for _, l := range s.Libraries {
				if !strings.Contains(pv, `src="../lib/`+filepath.Base(l.File)+`"`) {
					t.Errorf("%s/%s preview does not load %s", s.Slug, c.Name, l.File)
				}
				if strings.Index(pv, filepath.Base(l.File)) > strings.Index(pv, `src="../bundle.js"`) {
					t.Errorf("%s/%s: %s must load before the bundle", s.Slug, c.Name, l.File)
				}
			}
		}
	}

	filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !textual(p) {
			return nil
		}
		body := read(t, p)
		// Nothing may still point at the platform's asset store.
		if strings.Contains(body, "/_blob/") {
			t.Errorf("%s still contains /_blob/", p)
		}
		// Platform boilerplate (SKILL.md) is never published.
		if strings.Contains(body, "secret-boilerplate-marker") {
			t.Errorf("%s publishes the SKILL.md boilerplate", p)
		}
		return nil
	})
}

var (
	attrRe = regexp.MustCompile(`(?:href|src)="([^"]*)"`)
	urlRe  = regexp.MustCompile(`url\(\s*["']?([^"')]+)["']?\s*\)`)
	// data: urls are self-contained; what is inside them (an SVG's own #refs) is not a file.
	dataURLRe = regexp.MustCompile(`url\(\s*(?:"data:[^"]*"|'data:[^']*'|data:[^)]*)\s*\)`)
	jsRe      = regexp.MustCompile(`\.\./\.\./assets/[^"'\s)\\]+`)
)

func local(ref string) (string, bool) {
	if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "data:") ||
		strings.Contains(ref, "://") || strings.HasPrefix(ref, "mailto:") || strings.HasPrefix(ref, "//") {
		return "", false
	}
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	if u, err := url.PathUnescape(ref); err == nil {
		ref = u
	}
	return ref, ref != ""
}

// firstComponentDir is any component folder, as a base for paths written relative to a preview.
func firstComponentDir(t *testing.T, components string) string {
	t.Helper()
	entries, err := os.ReadDir(components)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != "lib" {
			return e.Name()
		}
	}
	t.Fatal("no component folders")
	return ""
}

// Every local link, script, stylesheet and CSS url() in the site must point at a file that exists.
func TestBuildHasNoBrokenLinks(t *testing.T) {
	out, _, _ := buildFixtures(t)
	checked := 0
	err := filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		var refs []string
		switch filepath.Ext(p) {
		case ".html":
			src := dataURLRe.ReplaceAllString(read(t, p), "url()")
			for _, m := range attrRe.FindAllStringSubmatch(src, -1) {
				refs = append(refs, m[1])
			}
			// url() is only a real reference inside CSS; rendered docs show it in code samples.
			if filepath.Base(p) == "preview.html" || filepath.Base(filepath.Dir(p)) == "specimens" {
				for _, m := range urlRe.FindAllStringSubmatch(src, -1) {
					refs = append(refs, m[1])
				}
			}
		case ".css":
			for _, m := range urlRe.FindAllStringSubmatch(dataURLRe.ReplaceAllString(read(t, p), "url()"), -1) {
				refs = append(refs, m[1])
			}
		case ".js":
			// Rewritten asset paths inside the bundle are relative to the previews that load it.
			if filepath.Base(p) == "bundle.js" {
				base := filepath.Join(filepath.Dir(p), firstComponentDir(t, filepath.Dir(p)))
				for _, m := range jsRe.FindAllString(read(t, p), -1) {
					if r, ok := local(m); ok {
						checked++
						if !exists(filepath.Join(base, r)) {
							t.Errorf("%s: bundle refers to missing %s", p, m)
						}
					}
				}
			}
		}
		for _, ref := range refs {
			r, ok := local(ref)
			if !ok {
				continue
			}
			checked++
			target := filepath.Join(filepath.Dir(p), filepath.FromSlash(r))
			if strings.HasSuffix(r, "/") {
				target = filepath.Join(target, "index.html")
			}
			if !exists(target) {
				t.Errorf("%s: broken reference %q", strings.TrimPrefix(p, out), ref)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The synthetic export alone has well over this many references; a low count means the
	// checker has stopped matching anything.
	if checked < 30 {
		t.Errorf("only %d references checked; the checker is probably not matching anything", checked)
	}
}

func TestBuildRefusesToClobber(t *testing.T) {
	_, systems, _ := buildFixtures(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "precious.txt"), []byte("keep"), 0o644)
	if _, err := Build(systems, dir); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Errorf("non-empty foreign dir: err = %v", err)
	}
	if !exists(filepath.Join(dir, "precious.txt")) {
		t.Error("foreign file was deleted")
	}
	if _, err := Build(systems, systems[0].Dir); err == nil {
		t.Error("building into a system folder should be refused")
	}
}

func TestBuildReplacesItsOwnOutput(t *testing.T) {
	_, systems, _ := buildFixtures(t)
	dir := filepath.Join(t.TempDir(), "site")
	if _, err := Build(systems, dir); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "stale.txt")
	os.WriteFile(stale, []byte("old"), 0o644)
	if _, err := Build(systems, dir); err != nil {
		t.Fatalf("rebuild into a folder this tool made: %v", err)
	}
	if exists(stale) {
		t.Error("a rebuild should clear files left from before")
	}
}

// What the synthetic export is built to exercise.
func TestMiniExport(t *testing.T) {
	out, _, mini := buildFixtures(t)
	root := filepath.Join(out, mini.Slug)

	// Component page: preview, README (summary, no raw HTML), props from the component's own .d.ts.
	page := read(t, filepath.Join(root, "components", "Button", "index.html"))
	for _, want := range []string{"<h1>Button</h1>", `src="preview.html"`, "<summary>Props</summary>",
		"ButtonProps", "The everyday button", `aria-current="page"`, `src="../../assets/Logos/mark.svg"`} {
		if !strings.Contains(page, want) {
			t.Errorf("Button page lacks %q", want)
		}
	}
	// Props for a component without its own .d.ts come from index.d.ts.
	if badge := read(t, filepath.Join(root, "components", "Badge", "index.html")); !strings.Contains(badge, "BadgeProps") {
		t.Error("Badge page should take its props from index.d.ts")
	}
	// A component with a README and no preview still gets a page, without a frame.
	notes := read(t, filepath.Join(root, "components", "Notes", "index.html"))
	if strings.Contains(notes, "<iframe") || !strings.Contains(notes, "guidance only") {
		t.Error("Notes page should show its README and no preview frame")
	}

	// Raw HTML in READMEs is dropped, never passed through.
	for _, p := range []string{filepath.Join(root, "components", "Button", "index.html"), filepath.Join(root, "index.html")} {
		if b := read(t, p); strings.Contains(b, "<script>alert") {
			t.Errorf("%s passes raw HTML through", p)
		}
	}

	// Asset URLs with a space are escaped in the preview that uses one.
	if b := read(t, filepath.Join(root, "components", "Badge", "preview.html")); !strings.Contains(b, `src="../../assets/Logos/wide%20mark.svg"`) {
		t.Error("Badge preview should point at the escaped asset path")
	}
	// The bundle's own string reference is rewritten relative to the previews that load it.
	if b := read(t, filepath.Join(root, "components", "bundle.js")); !strings.Contains(b, "url(../../assets/Logos/mark.svg)") {
		t.Error("bundle.js asset reference not rewritten")
	}
	// ...and the stylesheet's relative to itself.
	if b := read(t, filepath.Join(root, "components", "bundle.css")); !strings.Contains(b, "url(../assets/Logos/mark.svg)") {
		t.Error("bundle.css asset reference not rewritten")
	}

	// Two themes: tokens.css has a dark block that re-declares the alias, and the colour page shows both.
	css := read(t, filepath.Join(root, "tokens.css"))
	dark := css[strings.Index(css, `[data-theme="dark"]`):]
	for _, want := range []string{"--text: var(--ink);", "--ink: #ececee;"} {
		if !strings.Contains(dark, want) {
			t.Errorf("dark theme block lacks %q", want)
		}
	}
	spec := read(t, filepath.Join(root, "specimens", "colours.html"))
	if !strings.Contains(spec, `data-theme="dark"`) || strings.Contains(spec, "broken") {
		t.Error("colour specimen should show the dark theme and omit the invalid colour")
	}
	if scale := read(t, filepath.Join(root, "specimens", "spacing.html")); !strings.Contains(scale, `var(--space-1\.5)`) {
		t.Error("spacing specimen should escape the dotted token name")
	}

	// Extra guides become pages; SKILL.md and README.md do not.
	guide := read(t, filepath.Join(root, "docs", "guide.html"))
	if !strings.Contains(guide, "Using the Mini system") || !strings.Contains(guide, `src="../assets/Logos/mark.svg"`) {
		t.Error("guide page missing its content or rewritten image")
	}
	for _, stem := range []string{"SKILL", "README"} {
		if exists(filepath.Join(root, "docs", stem+".html")) {
			t.Errorf("%s.md must not become a page", stem)
		}
	}
	// The nav lists the guide by its title.
	if idx := read(t, filepath.Join(root, "index.html")); !strings.Contains(idx, "Using the Mini system") {
		t.Error("navigation should list the guide")
	}
}

func TestBrandBookPages(t *testing.T) {
	out, systems, _ := buildFixtures(t)
	for _, s := range systems {
		root := filepath.Join(out, s.Slug)
		for _, f := range []string{"colours.html", "typography.html", "spacing.html", "assets.html",
			"specimens/colours.html", "specimens/typography.html", "specimens/spacing.html"} {
			if !exists(filepath.Join(root, f)) {
				t.Errorf("%s: missing %s", s.Slug, f)
			}
		}
		// Every accepted colour appears once in the colour specimen, drawn from its own token.
		spec := read(t, filepath.Join(root, "specimens", "colours.html"))
		if got, want := strings.Count(spec, `class="dsx-tile"`), len(compileColours(s)); got != want {
			t.Errorf("%s: %d colour tiles, %d accepted colours", s.Slug, got, want)
		}
		if !strings.Contains(spec, `href="../tokens.css"`) {
			t.Errorf("%s: colour specimen does not load tokens.css", s.Slug)
		}
		// The overview carries the README; every page has the system navigation with the current page marked.
		if idx := read(t, filepath.Join(root, "index.html")); !strings.Contains(idx, `class="prose"`) {
			t.Errorf("%s: overview lacks the README", s.Slug)
		}
		for _, f := range []string{"index.html", "colours.html", "assets.html"} {
			if pg := read(t, filepath.Join(root, f)); !strings.Contains(pg, `aria-current="page"`) {
				t.Errorf("%s/%s: no current page in navigation", s.Slug, f)
			}
		}
		// Assets: one entry per file in the index.
		n := 0
		for _, g := range s.Assets {
			n += len(g.Files)
		}
		if got := strings.Count(read(t, filepath.Join(root, "assets.html")), `<span class="m">`); got != n {
			t.Errorf("%s: %d asset entries, want %d", s.Slug, got, n)
		}
		// Extra guides: a page each, and no docs/ folder when there are none.
		docs := extraDocs(s.Dir)
		for _, d := range docs {
			if !exists(filepath.Join(root, "docs", d.Stem+".html")) {
				t.Errorf("%s: missing docs/%s.html", s.Slug, d.Stem)
			}
		}
		if len(docs) == 0 && exists(filepath.Join(root, "docs")) {
			t.Errorf("%s has no extra guides, so no docs/ folder", s.Slug)
		}
	}
}

func TestColourGroupKey(t *testing.T) {
	for in, want := range map[string]string{"vt-green-700": "vt-green", "ws-forest-900": "ws-forest",
		"surface-page": "surface", "vt-ink": "vt", "plain": "plain", "alert-info-text": "alert"} {
		if got := colourGroupKey(in); got != want {
			t.Errorf("colourGroupKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstSentenceAndStripTitle(t *testing.T) {
	if got := firstSentence("# T\n\nThe **app's** `everyday` button. More text.\n\nSecond."); got != "The app's everyday button." {
		t.Errorf("got %q", got)
	}
	if got := stripTitle("# Button\n\nBody", "Button"); got != "Body" {
		t.Errorf("got %q", got)
	}
	if got := stripTitle("# Other\n\nBody", "Button"); got != "# Other\n\nBody" {
		t.Errorf("got %q", got)
	}
}

func TestInterfaceFrom(t *testing.T) {
	src := "export interface AProps {\n  a?: string;\n  nested?: { x: number };\n}\n" +
		"export interface BProps { b?: string }\n" +
		"export interface CProps {\n  c?: string;\n}\n"
	for name, want := range map[string]string{
		"AProps": "export interface AProps {\n  a?: string;\n  nested?: { x: number };\n}",
		"BProps": "export interface BProps { b?: string }",
		"CProps": "export interface CProps {\n  c?: string;\n}",
		"ZProps": "",
	} {
		if got := interfaceFrom(src, name); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}
