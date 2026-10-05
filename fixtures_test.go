package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joshmcarthur/design-export-docs/internal/fixtures"
	"github.com/joshmcarthur/design-export-docs/internal/system"
	"github.com/joshmcarthur/design-export-docs/internal/tokens"
)

var update = flag.Bool("update", false, "rewrite golden files in testdata/")

// The synthetic export contains one invalid colour ("broken") on purpose, to prove that bad tokens
// are dropped and reported. Real exports given in DED_FIXTURES must compile with no warnings.

func TestFixturesCheck(t *testing.T) {
	for _, f := range fixtures.All(t) {
		t.Run(f.Slug, func(t *testing.T) {
			if problems, err := checkOne(f.Dir, false); err != nil || problems != 0 {
				t.Fatalf("check: %d problems, err %v", problems, err)
			}
			problems, err := checkOne(f.Dir, true) // --strict also fails on dropped tokens
			if err != nil {
				t.Fatal(err)
			}
			if f.Mini && problems != 1 {
				t.Errorf("strict check of the synthetic export: %d problems, want exactly 1 (the invalid colour)", problems)
			}
			if !f.Mini && problems != 0 {
				t.Errorf("strict check: %d problems", problems)
			}
		})
	}
}

func TestTokensGolden(t *testing.T) {
	for _, f := range fixtures.All(t) {
		t.Run(f.Slug, func(t *testing.T) {
			s, err := system.Load(f.Dir, f.Slug)
			if err != nil {
				t.Fatal(err)
			}
			res := tokens.Compile(s.Tokens, tokens.Options{})
			if f.Mini {
				if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "broken") {
					t.Errorf("warnings = %v, want one about the invalid colour", res.Warnings)
				}
			} else if len(res.Warnings) != 0 {
				t.Errorf("warnings: %v", res.Warnings)
			}
			golden := filepath.Join("testdata", f.Slug+".tokens.css")
			if _, err := os.Stat(golden); err != nil && !f.Mini && !*update {
				return // no golden file for an external export
			}
			if *update {
				if err := os.WriteFile(golden, []byte(res.CSS), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run go test -update)", err)
			}
			if res.CSS != string(want) {
				t.Errorf("tokens.css differs from %s (run go test -update if the change is intended)", golden)
			}
		})
	}
}

// Every custom property a component uses must come from tokens.css or the component stylesheet,
// otherwise the generated site would render it unstyled. Bootstrap's own --bs-* variables are
// defined with fallbacks inside compiled Bootstrap CSS and are not checked.
func TestComponentsOnlyUseDefinedCustomProperties(t *testing.T) {
	use := regexp.MustCompile(`var\(--([A-Za-z0-9_-]+)`)
	for _, f := range fixtures.All(t) {
		t.Run(f.Slug, func(t *testing.T) {
			s, err := system.Load(f.Dir, f.Slug)
			if err != nil {
				t.Fatal(err)
			}
			css := tokens.Compile(s.Tokens, tokens.Options{}).CSS
			bundle, err := os.ReadFile(filepath.Join(f.Dir, "components", "bundle.css"))
			if err != nil {
				t.Fatal(err)
			}
			defined := func(name string) bool {
				return strings.Contains(css, "--"+name+":") || strings.Contains(string(bundle), "--"+name+":")
			}
			files, _ := filepath.Glob(filepath.Join(f.Dir, "components", "*", "preview.html"))
			files = append(files, filepath.Join(f.Dir, "components", "bundle.css"))
			for _, file := range files {
				b, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				for _, m := range use.FindAllStringSubmatch(string(b), -1) {
					if !strings.HasPrefix(m[1], "bs-") && !defined(m[1]) {
						t.Errorf("--%s is used in %s but defined nowhere", m[1], filepath.Base(filepath.Dir(file)))
					}
				}
			}
		})
	}
}
