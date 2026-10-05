# design-export-docs

Documentation sites from exported design-system folders.

It reads a folder of design-system files (tokens, components, assets) and writes a plain HTML
site: brand book, colours, typography, spacing, assets, and a live preview of every component.
Output is just files, so it can be hosted anywhere (GitHub Pages, Cloudflare Pages, S3, `file://`).

It reads the folder layout produced by exporting a Design System from Claude Design (claude.ai),
and no other format.

> Independent project. Not affiliated with, endorsed by, or sponsored by Anthropic. Claude and
> claude.ai are trademarks of Anthropic, PBC.

It exists because those exported files are not viewable on their own. Component previews expect
the platform to inject the stylesheet, fonts, React and the component bundle, and the CSS, JS and
previews refer to images by platform-only `/_blob/<id>` URLs. design-export-docs does that wiring.

## Install

**From a release.** Each [release](https://github.com/joshmcarthur/design-export-docs/releases) has
archives for macOS (Apple silicon and Intel), Linux (amd64 and arm64) and Windows (amd64), with a
`SHA256SUMS` file. Each is a single static binary of about 8 MB with no runtime dependencies.
Download one from the releases page, or with the GitHub CLI:

```bash
gh release download <tag> --repo joshmcarthur/design-export-docs --pattern '*darwin_arm64*'
tar -xzf design-export-docs_*_darwin_arm64.tar.gz     # then put the design-export-docs binary on your PATH
```

Check `design-export-docs version` afterwards, and verify the download against `SHA256SUMS`
(`grep darwin_arm64 SHA256SUMS | shasum -a 256 -c -`). See [Releasing](#releasing) for how
releases are made.

**From source.** You need Go 1.23 or later.

```bash
git clone https://github.com/joshmcarthur/design-export-docs
cd design-export-docs
go build -o design-export-docs .
```

This produces the same single binary: the page templates and viewer assets are embedded, and the
only dependency is [goldmark](https://github.com/yuin/goldmark) for rendering markdown. To put it
on your `PATH` instead, run `go install .`.

## Use

Build a site from one or more system folders, then look at it:

```bash
design-export-docs build --out dist ./my-system ./another-system
design-export-docs serve dist          # http://127.0.0.1:8080/
```

Each folder becomes a section of the site named after the folder. Give a different URL name with
`name=folder`:

```bash
design-export-docs build --out dist main=./my-system other=./another-system
```

### Commands

| Command | What it does |
| --- | --- |
| `design-export-docs build [--out dist] [name=]<dir>...` | Write the static site. |
| `design-export-docs serve [--addr 127.0.0.1:8080] [dir]` | Serve a built site locally. Does not rebuild; run `build` again after changes. |
| `design-export-docs check [--strict] <dir>...` | Validate exports without building: files exist and match the index, tokens compile, and every `/_blob/` reference resolves. `--strict` also fails on tokens that were dropped. |
| `design-export-docs tokens <dir>` | Print the compiled `tokens.css` for a system. |
| `design-export-docs version` | Print the version (`dev` for a build from source). |

`build` refuses to write into a folder that is not empty unless design-export-docs created it (it leaves a
`.design-export-docs-site` marker), and refuses to write into a system's own folder. It stops with an error
if anything in an export refers to an asset that the index does not know.

## Input

A system folder is the `project/` tree of an exported Design System. (A `SKILL.md` in the folder,
which exports contain, is ignored.) The pieces design-export-docs reads:

```
<system>/
├── design-system.json     index: v3, layout "files"; title, namespace, libraries, assets
├── tokens.json            colours (with themes), type, spacing, radius, shadow, other scales
├── README.md              the brand book (becomes the Overview page)
├── *.md                   any other top-level markdown becomes a page under docs/
├── components/
│   ├── bundle.js, bundle.css, index.d.ts, lib/
│   └── <Name>/            preview.html (+ README.md, <Name>.d.ts); Cover/ is the overview banner
├── assets/<Group>/        images and files named in the index (+ a README.md per group)
└── fonts/
```

Anything else in the index (a different `v` or `layout`) is rejected with a clear message.
Paths from the index are never trusted: anything with `..`, a leading `/` or a backslash is
refused, and symlinks in an export are not followed.

[`docs/export-format.md`](docs/export-format.md) explains how to make an export, and
[`testdata/mini`](testdata/mini) is a small example you can build and read.

## Output

```
dist/
├── index.html                 lists the systems
├── _viewer/                   the viewer's own CSS and JS
└── <system>/
    ├── index.html             overview: cover and README
    ├── colours.html, typography.html, spacing.html    token pages
    ├── assets.html            every asset, by group
    ├── components/index.html  grid of live previews, grouped by @dsCard
    ├── components/<Name>/     index.html (preview, README, props) and preview.html
    ├── docs/<name>.html       extra markdown guides
    ├── specimens/             the token pages' documents (they load tokens.css)
    ├── tokens.css, tokens.json
    └── fonts/, assets/, components/lib/ and any other top-level folders: copied as they are
```

What design-export-docs does to make an export work outside the platform it came from:

- **`tokens.css`** is compiled from `tokens.json` following the platform's rules: themes, aliases
  re-declared per theme, a class per type style, `@font-face` rules. Values the platform would not
  read are dropped and reported as warnings.
- **Previews** are wrapped so each loads `tokens.css`, `bundle.css`, React, ReactDOM and
  `bundle.js` in the order the platform's own viewer does, with `data-theme` set on `<html>`.
- **`/_blob/<id>` URLs** in the stylesheet, bundle, previews and READMEs are rewritten to relative
  paths to the files under `assets/`, so the site works under any base path.
- **Token pages** show each token through its own `tokens.css` inside an iframe, so one system's
  variables and classes never leak into the viewer or another system.

## Deploying

The output is static and uses relative URLs, so upload the folder anywhere, including under a
sub-path: GitHub Pages, Cloudflare Pages, S3, or any web server. Build in CI and publish `dist/`.

The site includes every file in the export, fonts and images among them, so publishing it makes
them public. Check that you have the right to do so first (commercial typefaces often do not allow
web redistribution).

## Releasing

Releases are automated with [release-please](https://github.com/googleapis/release-please). Write
commit messages in the [Conventional Commits](https://www.conventionalcommits.org/) style:

- `feat: ...` for a new feature (a minor version bump, or while below 1.0 also for breaking changes)
- `fix: ...` for a bug fix (a patch bump)
- `docs: ...` and `ci: ...` appear in the changelog but do not by themselves trigger a release
- add `!` after the type, or a `BREAKING CHANGE:` footer, for a breaking change

On every push to `main`, release-please opens or updates a **release pull request** that bumps the
version and writes `CHANGELOG.md` from those messages. **Merging it** creates the `vX.Y.Z` tag and
the GitHub release, and the same workflow then runs the tests, builds the archives and attaches
them to the release. Commits that do not follow the style are left out of the changelog and do not
trigger a release.

To force a particular version, put `Release-As: X.Y.Z` in the footer of a commit message.

To build the archives locally without releasing, run `./release.sh <version>`. It
writes `release/design-export-docs_<version>_<os>_<arch>.{tar.gz,zip}` and `SHA256SUMS`, and is the same
script the workflow runs.

## Development

```bash
go test ./...            # unit tests, plus tests that build the example exports
go test ./... -update    # rewrite testdata/*.tokens.css after an intended change to the compiler
```

The tests run against [`testdata/mini`](testdata/mini), a small synthetic export written for this
purpose. It has two themes and an alias, asset references in the stylesheet, bundle, previews and
READMEs, a library, a guide, a component with no preview, raw HTML in READMEs, an invalid colour
(which must be dropped and reported), and a stand-in for the platform's `SKILL.md` (which must
never be published). Tests check that no `/_blob/` URL survives and that every link, script,
stylesheet and CSS `url()` in the output points at a file that exists.

To run the same tests against real exports, list them in `DED_FIXTURES`, with absolute paths and
optional names, separated like `PATH`:

```bash
DED_FIXTURES="main=/abs/path/my-system:/abs/path/other" go test ./...
```

```
main.go, build.go, check.go    command line
release.sh                     cross-compiles and packages the release archives
testdata/mini                  synthetic export used by the tests
internal/fixtures              finds the export folders tests run against
internal/system                loads an export (index, assets, components, @dsCard markers)
internal/tokens                tokens.json -> tokens.css, plus structured entries for the token pages
internal/rewrite               /_blob/<id> -> exported file path
internal/markdown              README rendering (goldmark; raw HTML in READMEs is dropped)
internal/site                  build orchestration, preview wrapper, page templates, viewer assets
```

## Limitations

- Only the export layout above is supported. Anthropic does not publish a specification for it, so a
  change on their side can break this tool. Unknown format versions are rejected with an error
  instead of being guessed at.
- Previews run the component bundle exactly as exported. Treat an export like any code you
  would run in a browser: only build systems you trust.
- Preview frames resize to fit their content, which browsers block on `file://`; there they keep
  the height recorded in the preview's `@dsCard` marker.
- Platform limits (such as the maximum number of colours) are not enforced.

## Notice

This is an independent project. It is not affiliated with, endorsed by, or sponsored by Anthropic.
Claude and claude.ai are trademarks of Anthropic, PBC, mentioned here only to say which exports
the tool reads. It uses no Anthropic logos or branding.

## Licence

MIT. See [LICENSE](LICENSE).
