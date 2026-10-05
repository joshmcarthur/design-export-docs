// Package fixtures finds the exported design-system folders that tests run against.
//
// The built-in fixture is testdata/mini, a small synthetic export. To also run the tests against
// real exports, set DED_FIXTURES to a list of folders (separated like PATH), each optionally
// prefixed with a name: DED_FIXTURES="main=/path/to/my-system:/path/to/other".
package fixtures

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Fixture is one export folder.
type Fixture struct {
	Slug string
	Dir  string
	Mini bool // the built-in synthetic export, whose exact contents tests may rely on
}

// All returns the built-in fixture followed by any named in DED_FIXTURES.
func All(t testing.TB) []Fixture {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	mini := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "mini")
	out := []Fixture{{Slug: "mini", Dir: mini, Mini: true}}
	for _, entry := range filepath.SplitList(os.Getenv("DED_FIXTURES")) {
		slug, dir := "", entry
		if i := strings.Index(entry, "="); i > 0 {
			slug, dir = entry[:i], entry[i+1:]
		}
		if slug == "" {
			slug = filepath.Base(dir)
		}
		if _, err := os.Stat(filepath.Join(dir, "design-system.json")); err != nil {
			t.Fatalf("DED_FIXTURES: %s is not an export folder: %v", dir, err)
		}
		out = append(out, Fixture{Slug: slug, Dir: dir})
	}
	return out
}
