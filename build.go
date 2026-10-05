package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshmcarthur/design-export-docs/internal/site"
	"github.com/joshmcarthur/design-export-docs/internal/system"
)

func runBuild(args []string) error {
	fl := flag.NewFlagSet("build", flag.ContinueOnError)
	out := fl.String("out", "dist", "output folder (created, or replaced if design-export-docs made it)")
	if err := fl.Parse(args); err != nil {
		return err
	}
	if fl.NArg() == 0 {
		return errors.New("usage: design-export-docs build [--out dist] [name=]<system-dir>...")
	}
	var systems []*system.System
	for _, arg := range fl.Args() {
		slug, dir := "", arg
		if i := strings.Index(arg, "="); i > 0 {
			slug, dir = arg[:i], arg[i+1:]
		}
		s, err := system.Load(dir, slug)
		if err != nil {
			return err
		}
		systems = append(systems, s)
	}
	res, err := site.Build(systems, *out)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	fmt.Printf("wrote %d system(s) to %s\n", len(systems), *out)
	return nil
}

func runServe(args []string) error {
	fl := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fl.String("addr", "127.0.0.1:8080", "address to listen on")
	if err := fl.Parse(args); err != nil {
		return err
	}
	dir := "dist"
	if fl.NArg() > 0 {
		dir = fl.Arg(0)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return fmt.Errorf("%s has no index.html; run design-export-docs build first", dir)
	}
	fmt.Printf("serving %s at http://%s/\n", dir, *addr)
	return http.ListenAndServe(*addr, http.FileServer(http.Dir(dir)))
}
