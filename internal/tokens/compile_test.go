package tokens

import (
	"strings"
	"testing"
)

func compile(t *testing.T, src string) Result {
	t.Helper()
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return Compile(d, Options{})
}

func mustContain(t *testing.T, css string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(css, w) {
			t.Errorf("css missing %q\n--- css:\n%s", w, css)
		}
	}
}

func mustNotContain(t *testing.T, css string, bad ...string) {
	t.Helper()
	for _, b := range bad {
		if strings.Contains(css, b) {
			t.Errorf("css should not contain %q\n--- css:\n%s", b, css)
		}
	}
}

func TestThemesAliasesAndBorrowing(t *testing.T) {
	r := compile(t, `{
	  "color": {
	    "themes": [{"id":"Light"},{"id":"dark"}],
	    "tokens": [
	      {"name":"surface","value":{"light":"#FBF7F1","dark":"#1d1a17"}},
	      {"name":"text","value":"{ink}"},
	      {"name":"ink","value":{"light":"#2B2118","dark":"#f3ece3"}},
	      {"name":"only-dark","value":{"dark":"#000"}},
	      {"name":"plain","value":"rgb(1, 2, 3)"}
	    ]
	  }
	}`)
	mustContain(t, r.CSS,
		":root, [data-theme=\"light\"] {",
		"--surface: #fbf7f1;",
		"--text: var(--ink);",
		"--only-dark: #000;", // borrowed for the first theme
		"--plain: rgb(1, 2, 3);",
		"[data-theme=\"dark\"] {",
		"--surface: #1d1a17;",
		"--ink: #f3ece3;",
	)
	// The dark block must re-declare the alias or it keeps the light computed value.
	dark := r.CSS[strings.Index(r.CSS, "[data-theme=\"dark\"]"):]
	mustContain(t, dark, "--text: var(--ink);")
	if len(r.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", r.Warnings)
	}
}

func TestInvalidColoursAreDropped(t *testing.T) {
	r := compile(t, `{"color":{"themes":[{"id":"light"}],"tokens":[
	  {"name":"named","value":"red"},
	  {"name":"var","value":"var(--x)"},
	  {"name":"mix","value":"color-mix(in srgb, red, blue)"},
	  {"name":"self","value":"{self}"},
	  {"name":"missing","value":"{nope}"},
	  {"name":"a","value":"{b}"},
	  {"name":"b","value":"{a}"},
	  {"name":"bad name","value":"#fff"},
	  {"name":"ok","value":"#fff"},
	  {"name":"ok","value":"#000"}
	]}}`)
	mustContain(t, r.CSS, "--ok: #fff;")
	mustNotContain(t, r.CSS, "--named", "--var:", "--mix", "--self", "--missing", "--a:", "--b:", "bad name", "#000")
	if len(r.Warnings) < 8 {
		t.Errorf("expected a warning per dropped token, got %d: %v", len(r.Warnings), r.Warnings)
	}
}

func TestAliasChainDepthLimit(t *testing.T) {
	var toks []string
	toks = append(toks, `{"name":"c0","value":"#123456"}`)
	for i := 1; i <= 18; i++ {
		toks = append(toks, `{"name":"c`+itoa(i)+`","value":"{c`+itoa(i-1)+`}"}`)
	}
	r := compile(t, `{"color":{"themes":[{"id":"light"}],"tokens":[`+strings.Join(toks, ",")+`]}}`)
	mustContain(t, r.CSS, "--c15: var(--c14);")
	mustNotContain(t, r.CSS, "--c17:", "--c18:")
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func TestScalesFontsAndTypeStyles(t *testing.T) {
	r := compile(t, `{
	  "spacing":{"tokens":[{"name":"space-1.5","value":"6px"},{"name":"n","value":4},{"name":"bad","value":"calc(1px)"}]},
	  "radius":{"tokens":[{"name":"--r","value":"50%"}]},
	  "opacity":{"tokens":[{"name":"half","value":0.5}]},
	  "shadow":{"tokens":[{"name":"s","value":"0 1px 2px rgba(0,0,0,.1)"},{"name":"u","value":"url(x)"}]},
	  "type":{
	    "fonts":[{"family":"Acme","file":"A.woff2","weight":"300 800"},{"family":"Evil","file":"../x.woff2"}],
	    "families":{"sans":"\"Acme\", sans-serif","bad":"a; b"},
	    "groups":[{"name":"T","family":"sans","styles":[
	      {"name":"body.lg","fontSize":18,"lineHeight":1.5,"fontWeight":"bold","letterSpacing":"-0.01em","opticalSize":14},
	      {"name":"nosize","fontSize":"huge"}
	    ]}]
	  }
	}`)
	mustContain(t, r.CSS,
		`--space-1\.5: 6px;`, "--n: 4px;", "--r: 50%;", "--half: 0.5;",
		"--s: 0 1px 2px rgba(0,0,0,.1);",
		`--font-sans: "Acme", sans-serif;`,
		`src: url("fonts/A.woff2") format("woff2");`, "font-weight: 300 800;",
		`.body\.lg { font-family: var(--font-sans); font-size: 18px; line-height: 1.5; font-weight: 700; letter-spacing: -0.01em; font-variation-settings: 'opsz' 14; }`,
	)
	mustNotContain(t, r.CSS, "--bad", "calc", "--u:", "Evil", ".nosize", "--font-bad")
}

func TestFontBase(t *testing.T) {
	d, _ := Parse([]byte(`{"type":{"fonts":[{"family":"Acme","file":"fonts/A.ttf","weight":"400"}]}}`))
	r := Compile(d, Options{FontBase: "/x/"})
	mustContain(t, r.CSS, `url("/x/fonts/A.ttf") format("truetype")`)
}

func TestStructuredEntries(t *testing.T) {
	r := compile(t, `{
	  "color": {"themes":[{"id":"light"},{"id":"dark"}], "tokens":[
	    {"name":"ink","value":{"light":"#222","dark":"#eee"},"usage":"Body text."},
	    {"name":"text","value":"{ink}","usage":"Alias."},
	    {"name":"bad","value":"red"}
	  ]},
	  "spacing":{"tokens":[{"name":"s1","value":"4px","usage":"Gap."}]},
	  "shadow":{"tokens":[{"name":"lift","value":"0 1px 2px #000","usage":"Cards."}]},
	  "type":{"fonts":[{"family":"Acme","file":"A.woff2","weight":"400"}],
	    "families":{"sans":"Acme, sans-serif"},
	    "groups":[{"name":"Text","family":"sans","styles":[
	      {"name":"body","fontSize":16,"lineHeight":"24px","fontWeight":400,"usage":"Default.","sample":"Hello"}]}]}
	}`)
	if len(r.Colors) != 2 {
		t.Fatalf("colors = %+v", r.Colors)
	}
	text := r.Colors[1]
	// Aliases resolve to the target's literal value in each theme; hex shorthand is kept as written.
	if text.Name != "text" || text.Usage != "Alias." || len(text.Values) != 2 ||
		text.Values[0] != (ThemeValue{Theme: "light", Value: "#222", Alias: "ink"}) ||
		text.Values[1] != (ThemeValue{Theme: "dark", Value: "#eee", Alias: "ink"}) {
		t.Errorf("alias entry = %+v", text)
	}
	if len(r.Scales) != 2 || r.Scales[0].Family != "spacing" || r.Scales[0].Usage != "Gap." ||
		r.Scales[1].Family != "shadow" || r.Scales[1].Value != "0 1px 2px #000" {
		t.Errorf("scales = %+v", r.Scales)
	}
	if len(r.Fonts) != 1 || r.Fonts[0].File != "fonts/A.woff2" {
		t.Errorf("fonts = %+v", r.Fonts)
	}
	if len(r.Styles) != 1 || r.Styles[0] != (StyleEntry{Group: "Text", Name: "body", Family: "Acme, sans-serif",
		Size: "16px", LineHeight: "24px", Weight: "400", Usage: "Default.", Sample: "Hello"}) {
		t.Errorf("styles = %+v", r.Styles)
	}
	if len(r.Families) != 1 || r.Families[0].Key != "sans" {
		t.Errorf("families = %+v", r.Families)
	}
}
