// Package rewrite replaces claude.ai asset URLs in exported files.
//
// Components in the artifact refer to uploaded images as "/_blob/<id>", a path that only exists
// on claude.ai. In an export the same files live under assets/, and the index maps each blob id
// to its path, so the references can be pointed at the real files.
package rewrite

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var blobRe = regexp.MustCompile(`/_blob/([0-9a-f]{32})`)

// Rewriter replaces blob references with Prefix + the asset's path. Use a site-absolute Prefix
// (for example "/design/my-system/") so the result is valid inside CSS, JS strings and HTML
// at any depth.
type Rewriter struct {
	Paths  map[string]string // blob id -> path relative to the system folder
	Prefix string

	unresolved map[string]bool
	count      int
}

// New returns a Rewriter for the given blob map.
func New(paths map[string]string, prefix string) *Rewriter {
	return &Rewriter{Paths: paths, Prefix: prefix, unresolved: map[string]bool{}}
}

// Apply rewrites every blob reference in s. References to unknown blobs are left as they are
// and recorded in Unresolved.
func (r *Rewriter) Apply(s string) string {
	return blobRe.ReplaceAllStringFunc(s, func(m string) string {
		id := m[len("/_blob/"):]
		p, ok := r.Paths[id]
		if !ok {
			r.unresolved[id] = true
			return m
		}
		r.count++
		return r.Prefix + escapePath(p)
	})
}

// Count is the number of references rewritten so far.
func (r *Rewriter) Count() int { return r.count }

// Unresolved lists blob ids seen that the index does not know, sorted.
func (r *Rewriter) Unresolved() []string {
	out := make([]string, 0, len(r.unresolved))
	for id := range r.unresolved {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Refs returns the distinct blob ids referenced in s.
func Refs(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range blobRe.FindAllStringSubmatch(s, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}
