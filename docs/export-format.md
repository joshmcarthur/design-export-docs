# Making an export

design-export-docs reads a folder in the layout produced by exporting a Design System from Claude
Design (claude.ai). Anthropic does not publish a specification for it, so this page describes what
the tool relies on, not an official format. See [`testdata/mini`](../testdata/mini) for a small
complete example.

## The folder

It is the design system's `project/` tree:

| Path | What it is |
| --- | --- |
| `design-system.json` | The index. Must have `"v": 3` and `"layout": "files"`; names the title, bundle namespace, libraries, and every uploaded asset. |
| `tokens.json` | Colours (with themes), type, spacing, radius, shadow and any other scales. Each family is a list of `{name, value, usage}`. |
| `README.md` | The brand book. Other top-level `*.md` files become guide pages. |
| `components/` | `bundle.js` and `bundle.css`, `index.d.ts`, `lib/` (libraries such as React), and one folder per component with `preview.html`, `README.md` and `<Name>.d.ts`. A preview's first line is a `<!-- @dsCard group="..." height=N -->` marker. |
| `assets/<Group>/` | Images and other files, plus a `README.md` per group. |
| `fonts/` | Font files named by `tokens.json`. |

## Getting the files

1. Copy the design system's text files (everything under `project/`) into a folder.
2. Binary assets are not in that file list: they live in the project's asset store, and
   `design-system.json` records each one under `assetGroups.<Group>.files`, with a `name` (path
   below the group folder; bytes outside `A-Za-z0-9_./-` are written `~` plus two hex digits in the
   key), a `blob` id, a `size` and a `type`. Save each asset to `assets/<Group>/<name>`.
3. Run `design-export-docs check <folder>`. It reports a missing asset, a size that disagrees with
   the index, and any `/_blob/<id>` reference in the files that the index does not account for.

The files refer to uploaded images as `/_blob/<id>`, which only works on the platform. The tool
rewrites those to the asset files from step 2, so step 2 matters: without the assets, the site has
broken images.

## What to leave out

Every export also contains the platform's own runtime: a `SKILL.md` of instructions for agents, an
`index.html` viewer, and an `artifact-type/` folder. They are not part of the design system, they
depend on the platform, and no licence for them is published. The tool ignores a `SKILL.md` if it
finds one. Do not commit these files if you share your export.

## Stability

The index carries a version (`v`), the bundle header a `format`, and the runtime a contract
version. Anthropic has not promised any of them stay stable. The tool rejects an index it does not
recognise instead of guessing, so a change upstream shows up as an error, not a wrong site.
