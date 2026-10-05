package site

import (
	"strings"
	"testing"
)

var testWrap = Wrap{
	TokensCSS: "../../tokens.css",
	BundleCSS: "../bundle.css",
	Libraries: []string{"../lib/react.js", "../lib/react-dom.js"},
	Bundle:    "../bundle.js",
	Theme:     "light",
}

func indexOf(t *testing.T, s, sub string) int {
	t.Helper()
	i := strings.Index(s, sub)
	if i < 0 {
		t.Fatalf("missing %q in:\n%s", sub, s)
	}
	return i
}

func TestWrapPreviewKeepsMarkerAndOrdersAssets(t *testing.T) {
	doc := "<!-- @dsCard group=\"A\" height=88 -->\n<!doctype html>\n<html lang=\"en\"><head><meta charset=\"utf-8\"><title>x</title></head>\n<body><script>var E=window.Ns</script></body></html>"
	got := WrapPreview(doc, testWrap)
	if !strings.HasPrefix(got, "<!-- @dsCard group=\"A\" height=88 -->\n<!doctype html>") {
		t.Errorf("marker/doctype not preserved:\n%s", got)
	}
	if !strings.Contains(got, `<html lang="en" data-theme="light">`) {
		t.Errorf("data-theme not added:\n%s", got)
	}
	order := []string{`href="../../tokens.css"`, `href="../bundle.css"`, "URLSearchParams",
		`src="../lib/react.js"`, `src="../lib/react-dom.js"`, `src="../bundle.js"`, "var E=window.Ns"}
	last := -1
	for _, o := range order {
		i := indexOf(t, got, o)
		if i < last {
			t.Errorf("%q is out of order", o)
		}
		last = i
	}
	if strings.Index(got, "<meta charset") > strings.Index(got, `href="../../tokens.css"`) {
		t.Error("assets should follow the existing head content's opening tag, after <head>")
	}
}

func TestWrapPreviewVariants(t *testing.T) {
	// A preview that sets its own theme keeps it.
	got := WrapPreview(`<html data-theme="dark"><head></head><body></body></html>`, testWrap)
	if strings.Contains(got, `data-theme="light"`) || !strings.Contains(got, `data-theme="dark"`) {
		t.Errorf("own theme not kept:\n%s", got)
	}
	// No <head>: one is created.
	got = WrapPreview(`<html><body>hi</body></html>`, testWrap)
	if !strings.Contains(got, "<head><link") || !strings.Contains(got, "hi") {
		t.Errorf("head not created:\n%s", got)
	}
	// A fragment is wrapped in a document.
	got = WrapPreview(`<p>frag</p>`, testWrap)
	if !strings.HasPrefix(got, "<!doctype html><html data-theme=\"light\">") || !strings.Contains(got, "<p>frag</p></body>") {
		t.Errorf("fragment not wrapped:\n%s", got)
	}
	// No bundle and no libraries: a system that does not use React.
	got = WrapPreview(`<html><head></head><body></body></html>`, Wrap{TokensCSS: "t.css"})
	if strings.Contains(got, "<script src") || strings.Contains(got, "<html data-theme") {
		t.Errorf("unexpected assets:\n%s", got)
	}
}
