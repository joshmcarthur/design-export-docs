package site

import (
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/joshmcarthur/design-export-docs/internal/markdown"
	"github.com/joshmcarthur/design-export-docs/internal/rewrite"
	"github.com/joshmcarthur/design-export-docs/internal/system"
	"github.com/joshmcarthur/design-export-docs/internal/tokens"
)

// Keys of the sections in a system's navigation.
const (
	keyOverview   = "overview"
	keyColours    = "colours"
	keyTypography = "typography"
	keySpacing    = "spacing"
	keyAssets     = "assets"
	keyComponents = "components"
)

// extraDoc is a top-level markdown file other than README.md: further guides, shown as pages.
type extraDoc struct{ Stem, Title, File string }

var extraNameRe = regexp.MustCompile(`^([A-Za-z0-9._-]+)\.md$`)

func extraDocs(dir string) []extraDoc {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []extraDoc
	for _, e := range entries {
		m := extraNameRe.FindStringSubmatch(e.Name())
		if e.IsDir() || m == nil || strings.HasPrefix(e.Name(), ".") || e.Name() == "README.md" || e.Name() == "SKILL.md" {
			continue
		}
		title := m[1]
		if b, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			if t := titleRe.FindStringSubmatch(string(b)); t != nil {
				title = strings.TrimSpace(t[1])
			}
		}
		out = append(out, extraDoc{Stem: m[1], Title: title, File: e.Name()})
	}
	return out
}

func navPages(s *system.System) []pageLink {
	pages := []pageLink{
		{"Overview", "index.html", keyOverview},
		{"Colours", "colours.html", keyColours},
		{"Typography", "typography.html", keyTypography},
		{"Spacing and shape", "spacing.html", keySpacing},
	}
	if len(s.Assets) > 0 {
		pages = append(pages, pageLink{"Assets", "assets.html", keyAssets})
	}
	pages = append(pages, pageLink{"Components", "components/index.html", keyComponents})
	for _, d := range extraDocs(s.Dir) {
		pages = append(pages, pageLink{d.Title, "docs/" + d.Stem + ".html", "doc:" + d.Stem})
	}
	return pages
}

// readMarkdown renders a file under the system folder, pointing claude.ai asset urls at the
// exported files. prefix is the path from the page to the system folder. A missing file is not an error.
func (b *builder) readMarkdown(s *system.System, rel, prefix string, strip string) (template.HTML, error) {
	raw, err := os.ReadFile(filepath.Join(s.Dir, filepath.FromSlash(rel)))
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	rw := rewrite.New(s.BlobPaths(), prefix)
	text := rw.Apply(string(raw))
	if strip != "" {
		text = stripTitle(text, strip)
	}
	html, err := markdown.Render(text)
	if err != nil {
		b.res.Warnings = append(b.res.Warnings, fmt.Sprintf("%s: %s: %v", s.Slug, rel, err))
	}
	return html, nil
}

func (b *builder) renderSystem(s *system.System, v *systemView) error {
	res := tokens.Compile(s.Tokens, tokens.Options{})
	base := page{System: v}
	sysPage := func(title, section string) page {
		p := base
		p.Title, p.Root, p.SysRoot, p.Section = title+" · "+s.Title, up(1), "", section
		return p
	}

	// Overview: the cover and the brand book.
	readme, err := b.readMarkdown(s, "README.md", "", "")
	if err != nil {
		return err
	}
	p := sysPage("Overview", keyOverview)
	p.Title = s.Title
	p.Content = struct {
		Title    string
		HasCover bool
		Readme   template.HTML
	}{s.Title, v.HasCover, readme}
	if err := b.render("system.html", path.Join(s.Slug, "index.html"), p); err != nil {
		return err
	}

	// Token pages: each shows a specimen document that uses the system's own tokens.css.
	type specimenPage struct {
		Heading, Lede, Src string
		Count              int
		TokensJSON         bool
	}
	for _, sp := range []struct {
		key, file, heading, lede string
		count                    int
		data                     any
		tmpl                     string
	}{
		{keyColours, "colours", "Colours", "Each colour as the tokens define it, in every theme.", len(res.Colors), newColorsData(res), "specimen_colours.html"},
		{keyTypography, "typography", "Typography", "Font families, faces and text styles.", len(res.Styles), newTypeData(res), "specimen_typography.html"},
		{keySpacing, "spacing", "Spacing and shape", "Spacing, radii, shadows and any other scales.", len(res.Scales), newScaleData(res), "specimen_scale.html"},
	} {
		if err := b.renderSpecimen(sp.tmpl, path.Join(s.Slug, "specimens", sp.file+".html"), sp.data); err != nil {
			return err
		}
		pg := sysPage(sp.heading, sp.key)
		pg.Content = specimenPage{sp.heading, sp.lede, "specimens/" + sp.file + ".html", sp.count, sp.key == keyColours}
		if err := b.render("specimen.html", path.Join(s.Slug, sp.file+".html"), pg); err != nil {
			return err
		}
	}

	// Assets.
	if len(s.Assets) > 0 {
		ad, err := b.assetsData(s)
		if err != nil {
			return err
		}
		pg := sysPage("Assets", keyAssets)
		pg.Content = ad
		if err := b.render("assets.html", path.Join(s.Slug, "assets.html"), pg); err != nil {
			return err
		}
	}

	// Further guides.
	for _, d := range extraDocs(s.Dir) {
		body, err := b.readMarkdown(s, d.File, "../", "")
		if err != nil {
			return err
		}
		pg := base
		pg.Title, pg.Root, pg.SysRoot, pg.Section = d.Title+" · "+s.Title, up(2), up(1), "doc:"+d.Stem
		pg.Content = struct {
			Title string
			Body  template.HTML
		}{d.Title, body}
		if err := b.render("doc.html", path.Join(s.Slug, "docs", d.Stem+".html"), pg); err != nil {
			return err
		}
	}

	// Component grid and one page per component.
	pg := base
	pg.Title, pg.Root, pg.SysRoot, pg.Section, pg.ShowSide = "Components · "+s.Title, up(2), up(1), keyComponents, true
	pg.Content = v
	if err := b.render("components.html", path.Join(s.Slug, "components", "index.html"), pg); err != nil {
		return err
	}
	indexTS, _ := os.ReadFile(filepath.Join(s.Dir, "components", "index.d.ts"))
	for _, c := range s.Components {
		if c.Name == "Cover" {
			continue
		}
		cc := componentContent{Name: c.Name, Group: c.Card.Group, Height: 120, HasPreview: c.HasPreview}
		for _, g := range v.Groups {
			for _, cv := range g.Cards {
				if cv.Name == c.Name {
					cc.Height = cv.Height
				}
			}
		}
		if c.HasReadme {
			raw, err := os.ReadFile(filepath.Join(s.Dir, "components", c.Name, "README.md"))
			if err != nil {
				return err
			}
			rw := rewrite.New(s.BlobPaths(), "../../")
			text := stripTitle(rw.Apply(string(raw)), c.Name)
			cc.Summary = firstSentence(text)
			html, err := markdown.Render(text)
			if err != nil {
				b.res.Warnings = append(b.res.Warnings, fmt.Sprintf("%s: components/%s/README.md: %v", s.Slug, c.Name, err))
			}
			cc.Readme = html
		}
		cc.Props = props(s.Dir, c.Name, string(indexTS))

		cp := base
		cp.Title, cp.Root, cp.SysRoot, cp.Current = c.Name+" · "+s.Title, up(3), up(2), c.Name
		cp.Section, cp.ShowSide = keyComponents, true
		cp.Content = cc
		if err := b.render("component.html", path.Join(s.Slug, "components", c.Name, "index.html"), cp); err != nil {
			return err
		}
	}
	return nil
}

// renderSpecimen writes a standalone document (no viewer chrome) that links the system's tokens.css.
func (b *builder) renderSpecimen(tmplFile, outPath string, data any) error {
	t, err := template.ParseFS(assets, "templates/specimen_base.html", "templates/"+tmplFile)
	if err != nil {
		return err
	}
	var buf strings.Builder
	if err := t.ExecuteTemplate(&buf, "specimen", data); err != nil {
		return fmt.Errorf("%s: %w", tmplFile, err)
	}
	return b.write(outPath, []byte(buf.String()))
}

// ---- specimen data ----
//
// Styles are built from token names the compiler has already validated ([A-Za-z0-9_.-], dots
// escaped), so they are safe to mark as CSS.

func cssIdent(n string) string { return strings.ReplaceAll(n, ".", `\.`) }

func varStyle(prop, name string) template.CSS {
	return template.CSS(prop + ":var(--" + cssIdent(name) + ")")
}

type colorsData struct {
	Multi  bool
	Themes []tokens.Theme
	Groups []colorGroup
}

type colorGroup struct {
	Name  string
	Tiles []colorTile
}

type colorTile struct {
	Name     string
	Usage    string
	Swatches []swatch
}

type swatch struct {
	Theme, Value, Alias string
	Style               template.CSS
}

var numSuffix = regexp.MustCompile(`-\d+$`)

// colourGroupKey groups a colour with its scale ("vt-green-700" -> "vt-green"); other names group
// by their first segment ("surface-page" -> "surface").
func colourGroupKey(n string) string {
	if numSuffix.MatchString(n) {
		return numSuffix.ReplaceAllString(n, "")
	}
	if i := strings.Index(n, "-"); i > 0 {
		return n[:i]
	}
	return n
}

func newColorsData(res tokens.Result) colorsData {
	d := colorsData{Multi: len(res.Themes) > 1, Themes: res.Themes}
	idx := map[string]int{}
	for _, c := range res.Colors {
		t := colorTile{Name: c.Name, Usage: c.Usage}
		for _, v := range c.Values {
			t.Swatches = append(t.Swatches, swatch{Theme: v.Theme, Value: v.Value, Alias: v.Alias,
				Style: varStyle("background", c.Name)})
		}
		k := colourGroupKey(c.Name)
		i, ok := idx[k]
		if !ok {
			i = len(d.Groups)
			idx[k] = i
			d.Groups = append(d.Groups, colorGroup{Name: k})
		}
		d.Groups[i].Tiles = append(d.Groups[i].Tiles, t)
	}
	return d
}

type typeData struct {
	Families []familyView
	Faces    []faceGroup
	Groups   []styleGroup
}

type familyView struct{ Key, Stack string }

type faceGroup struct {
	Family string
	Faces  []faceView
}

type faceView struct {
	File, Weight, Style string
	Sample              template.CSS // empty if the family name is not a plain name
}

type styleGroup struct {
	Name   string
	Styles []styleView
}

type styleView struct{ Name, Meta, Usage, Sample string }

var plainFamilyRe = regexp.MustCompile(`^[A-Za-z0-9 _.-]+$`)

const defaultSample = "The quick brown fox jumps over the lazy dog"

func newTypeData(res tokens.Result) typeData {
	var d typeData
	for _, f := range res.Families {
		d.Families = append(d.Families, familyView{f.Key, f.Stack})
	}
	gi := map[string]int{}
	for _, f := range res.Fonts {
		i, ok := gi[f.Family]
		if !ok {
			i = len(d.Faces)
			gi[f.Family] = i
			d.Faces = append(d.Faces, faceGroup{Family: f.Family})
		}
		fv := faceView{File: f.File, Weight: f.Weight, Style: f.Style}
		if plainFamilyRe.MatchString(f.Family) {
			fv.Sample = template.CSS(fmt.Sprintf(`font-family:"%s";font-weight:%s;font-style:%s`, f.Family, f.Weight, f.Style))
		}
		d.Faces[i].Faces = append(d.Faces[i].Faces, fv)
	}
	si := map[string]int{}
	for _, st := range res.Styles {
		i, ok := si[st.Group]
		if !ok {
			i = len(d.Groups)
			si[st.Group] = i
			d.Groups = append(d.Groups, styleGroup{Name: st.Group})
		}
		meta := []string{st.Size}
		if st.LineHeight != "" {
			meta[0] += " / " + st.LineHeight
		}
		if st.Weight != "" {
			meta = append(meta, "weight "+st.Weight)
		}
		if st.LetterSpacing != "" {
			meta = append(meta, "tracking "+st.LetterSpacing)
		}
		sample := st.Sample
		if sample == "" {
			sample = defaultSample
		}
		d.Groups[i].Styles = append(d.Groups[i].Styles, styleView{st.Name, strings.Join(meta, " · "), st.Usage, sample})
	}
	return d
}

type scaleData struct {
	Spacing, Radius, Shadow []scaleView
	Others                  []otherFamily
}

type scaleView struct {
	Name, Value, Usage string
	Style              template.CSS
}

type otherFamily struct {
	Key    string
	Tokens []scaleView
}

func newScaleData(res tokens.Result) scaleData {
	var d scaleData
	oi := map[string]int{}
	for _, e := range res.Scales {
		v := scaleView{Name: e.Name, Value: e.Value, Usage: e.Usage}
		switch e.Family {
		case "spacing":
			v.Style = varStyle("width", e.Name)
			d.Spacing = append(d.Spacing, v)
		case "radius":
			v.Style = varStyle("border-radius", e.Name)
			d.Radius = append(d.Radius, v)
		case "shadow":
			v.Style = varStyle("box-shadow", e.Name)
			d.Shadow = append(d.Shadow, v)
		default:
			i, ok := oi[e.Family]
			if !ok {
				i = len(d.Others)
				oi[e.Family] = i
				d.Others = append(d.Others, otherFamily{Key: e.Family})
			}
			d.Others[i].Tokens = append(d.Others[i].Tokens, v)
		}
	}
	sort.SliceStable(d.Others, func(i, j int) bool { return d.Others[i].Key < d.Others[j].Key })
	return d
}

// ---- assets ----

type assetsData struct{ Groups []assetGroupView }

type assetGroupView struct {
	Name   string
	Readme template.HTML
	Files  []assetView
}

type assetView struct {
	Name, Path, Size, Type string
	Image                  bool
}

func (b *builder) assetsData(s *system.System) (assetsData, error) {
	var d assetsData
	for _, g := range s.Assets {
		gv := assetGroupView{Name: g.Name}
		var err error
		if gv.Readme, err = b.readMarkdown(s, "assets/"+g.Name+"/README.md", "", ""); err != nil {
			return d, err
		}
		for _, a := range g.Files {
			gv.Files = append(gv.Files, assetView{Name: a.Name, Path: a.Path, Size: humanSize(a.Size), Type: a.Type,
				Image: strings.HasPrefix(a.Type, "image/")})
		}
		d.Groups = append(d.Groups, gv)
	}
	return d, nil
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
