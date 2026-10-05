package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshmcarthur/design-export-docs/internal/rewrite"
	"github.com/joshmcarthur/design-export-docs/internal/system"
	"github.com/joshmcarthur/design-export-docs/internal/tokens"
)

func runTokens(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: design-export-docs tokens <system-dir>")
	}
	s, err := system.Load(args[0], "")
	if err != nil {
		return err
	}
	res := tokens.Compile(s.Tokens, tokens.Options{})
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	fmt.Print(res.CSS)
	return nil
}

func runCheck(args []string) error {
	fl := flag.NewFlagSet("check", flag.ContinueOnError)
	strict := fl.Bool("strict", false, "treat token warnings as errors")
	if err := fl.Parse(args); err != nil {
		return err
	}
	if fl.NArg() == 0 {
		return errors.New("usage: design-export-docs check [--strict] <system-dir>...")
	}
	failed := false
	for _, dir := range fl.Args() {
		problems, err := checkOne(dir, *strict)
		if err != nil {
			return err
		}
		if problems > 0 {
			failed = true
		}
	}
	if failed {
		return errors.New("check failed")
	}
	return nil
}

func checkOne(dir string, strict bool) (int, error) {
	s, err := system.Load(dir, "")
	if err != nil {
		return 0, err
	}
	problems := 0
	fail := func(format string, a ...any) {
		problems++
		fmt.Printf("  error: "+format+"\n", a...)
	}

	previews := 0
	for _, c := range s.Components {
		if c.HasPreview {
			previews++
		}
	}
	fmt.Printf("%s (%s, namespace %q)\n", s.Title, s.Slug, s.Namespace)
	fmt.Printf("  components: %d (%d with previews)\n", len(s.Components), previews)

	res := tokens.Compile(s.Tokens, tokens.Options{})
	fmt.Printf("  tokens: %d colours, %d shadows, %d spacing, %d radii, %d fonts, %d type groups\n",
		len(s.Tokens.Colors), len(s.Tokens.Shadows), len(s.Tokens.Spacing), len(s.Tokens.Radius),
		len(s.Tokens.Type.Fonts), len(s.Tokens.Type.Groups))
	for _, w := range res.Warnings {
		if strict {
			fail("tokens: %s", w)
		} else {
			fmt.Printf("  warning: tokens: %s\n", w)
		}
	}

	// Files named by the index and tokens must exist.
	assets := 0
	for _, g := range s.Assets {
		for _, a := range g.Files {
			assets++
			st, err := os.Stat(filepath.Join(s.Dir, filepath.FromSlash(a.Path)))
			switch {
			case errors.Is(err, fs.ErrNotExist):
				fail("asset missing: %s", a.Path)
			case err != nil:
				fail("asset %s: %v", a.Path, err)
			case st.Size() != a.Size:
				fail("asset %s is %d bytes, index says %d", a.Path, st.Size(), a.Size)
			}
		}
	}
	fmt.Printf("  assets: %d in %d groups\n", assets, len(s.Assets))
	for _, f := range s.Tokens.Type.Fonts {
		p := f.File
		if !strings.Contains(p, "/") {
			p = "fonts/" + p
		}
		if !system.SafeRel(p) {
			fail("unsafe font path: %q", f.File)
			continue
		}
		if _, err := os.Stat(filepath.Join(s.Dir, filepath.FromSlash(p))); err != nil {
			fail("font file missing: %s", p)
		}
	}
	for _, l := range s.Libraries {
		if _, err := os.Stat(filepath.Join(s.Dir, filepath.FromSlash(l.File))); err != nil {
			fail("library file missing: %s", l.File)
		}
	}

	// Every claude.ai blob reference in a text file must map to an asset.
	paths := s.BlobPaths()
	rw := rewrite.New(paths, "/")
	refs := map[string]bool{}
	err = filepath.WalkDir(s.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isText(p) {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, id := range rewrite.Refs(string(b)) {
			refs[id] = true
		}
		rw.Apply(string(b))
		return nil
	})
	if err != nil {
		return problems, err
	}
	fmt.Printf("  blob references: %d distinct, %d unresolved\n", len(refs), len(rw.Unresolved()))
	for _, id := range rw.Unresolved() {
		fail("blob %s is referenced but not in the index", id)
	}
	return problems, nil
}

func isText(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".css", ".js", ".html", ".md", ".json", ".svg", ".ts":
		return !strings.HasSuffix(p, "design-system.json") && !strings.HasSuffix(p, "SKILL.md")
	}
	return false
}
