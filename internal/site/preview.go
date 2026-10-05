package site

import (
	"regexp"
	"strings"
)

// Wrap says what a component preview needs around it. On claude.ai the viewer's frame preloads
// these; a static page has to carry them itself. All paths are relative to the preview file.
type Wrap struct {
	TokensCSS string   // compiled tokens.css
	BundleCSS string   // component stylesheet; empty if the system has none
	Libraries []string // scripts to load before the bundle, in order (React, ReactDOM)
	Bundle    string   // components/bundle.js; empty if the system has none
	Theme     string   // initial data-theme, the system's first theme
}

var (
	htmlTagRe   = regexp.MustCompile(`(?is)<html(\s[^>]*)?>`)
	headTagRe   = regexp.MustCompile(`(?is)<head(\s[^>]*)?>`)
	charsetRe   = regexp.MustCompile(`(?is)<meta\s[^>]*charset[^>]*>`)
	dataThemeRe = regexp.MustCompile(`(?i)\sdata-theme\s*=`)
)

// themeSnippet lets the viewer choose a theme with ?theme=<id> without any other machinery.
const themeSnippet = `<script>(function(){var t=new URLSearchParams(location.search).get("theme");` +
	`if(t&&/^[A-Za-z0-9_.-]+$/.test(t))document.documentElement.setAttribute("data-theme",t)})()</script>`

// WrapPreview makes a component's preview.html self-contained: it sets data-theme on <html>
// and adds the tokens, stylesheet, libraries and bundle ahead of the preview's own markup, in
// the order the platform's frame loads them. The marker comment on line 1 is kept.
func WrapPreview(doc string, w Wrap) string {
	var inj strings.Builder
	inj.WriteString(`<link rel="stylesheet" href="` + w.TokensCSS + `">`)
	if w.BundleCSS != "" {
		inj.WriteString(`<link rel="stylesheet" href="` + w.BundleCSS + `">`)
	}
	inj.WriteString(themeSnippet)
	for _, l := range w.Libraries {
		inj.WriteString(`<script src="` + l + `"></script>`)
	}
	if w.Bundle != "" {
		inj.WriteString(`<script src="` + w.Bundle + `"></script>`)
	}

	attr := ""
	if w.Theme != "" {
		attr = ` data-theme="` + w.Theme + `"`
	}

	// <html>: add data-theme unless the preview sets its own.
	if loc := htmlTagRe.FindStringIndex(doc); loc != nil {
		tag := doc[loc[0]:loc[1]]
		if attr != "" && !dataThemeRe.MatchString(tag) {
			tag = strings.TrimSuffix(tag, ">") + attr + ">"
			doc = doc[:loc[0]] + tag + doc[loc[1]:]
		}
	} else {
		// A bare fragment: give it a document to live in.
		return "<!doctype html><html" + attr + "><head><meta charset=\"utf-8\">" + inj.String() +
			"</head><body>" + doc + "</body></html>"
	}

	if loc := headTagRe.FindStringIndex(doc); loc != nil {
		// The charset declaration must stay within the first 1024 bytes, so go after it.
		at := loc[1]
		if end := strings.Index(strings.ToLower(doc), "</head"); end < 0 || at < end {
			if cs := charsetRe.FindStringIndex(doc[at:]); cs != nil && (end < 0 || at+cs[1] <= end) {
				at += cs[1]
			}
		}
		return doc[:at] + inj.String() + doc[at:]
	}
	loc := htmlTagRe.FindStringIndex(doc)
	return doc[:loc[1]] + "<head>" + inj.String() + "</head>" + doc[loc[1]:]
}
