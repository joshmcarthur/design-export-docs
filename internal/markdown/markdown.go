// Package markdown renders design system READMEs.
package markdown

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	// Raw HTML is left out (goldmark's default): README text is written by the artifact's
	// authors and is treated as untrusted. Markup examples live in code fences and still show.
)

// Render converts GitHub-flavoured markdown to HTML. Raw HTML in the source is omitted.
func Render(src string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil // #nosec G203 -- goldmark safe mode, no raw HTML
}
