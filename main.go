// Command design-export-docs builds documentation sites from exported design-system folders.
package main

import (
	"fmt"
	"os"
)

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `design-export-docs - documentation sites from exported design-system folders

Usage:
  design-export-docs build [--out dist] [name=]<system-dir>...   write the static site
  design-export-docs serve [--addr 127.0.0.1:8080] [dir]         serve a built site locally
  design-export-docs check [--strict] <system-dir>...            validate exports (files, tokens, asset references)
  design-export-docs tokens <system-dir>                         print the compiled tokens.css
  design-export-docs version                                     print the version

A <system-dir> is a folder containing design-system.json, tokens.json, components/ and assets/,
as produced by exporting a Design System from Claude Design (claude.ai).

Independent project, not affiliated with, endorsed by, or sponsored by Anthropic.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = runBuild(os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:])
	case "check":
		err = runCheck(os.Args[2:])
	case "tokens":
		err = runTokens(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("design-export-docs", version)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "design-export-docs: unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "design-export-docs:", err)
		os.Exit(1)
	}
}
