# report-gen-pdf

Publishable Marist / OCB institutional-benchmark PDFs from the same processed
IPEDS extracts that back `dashboard-primitives`.

## Requirements

- Go 1.22+
- [`typst`](https://typst.app/) on `PATH`

Templates, fonts, and letterhead are embedded in the binary. The first compile
downloads `@preview/cetz` (needs network); later runs use the Typst cache.

## CLI

```bash
# from dashboard extracts
go run ./cmd/reportgen \
  -data ../dashboard-primitives/data/processed \
  -year 2023 \
  -out out/report.pdf

# also write an editable Typst project
go run ./cmd/reportgen \
  -data ../dashboard-primitives/data/processed \
  -year 2023 \
  -out out/report.pdf \
  -typst-dir out/report-typst

# hand-authored JSON
go run ./cmd/reportgen -in examples/sample.json -out out/sample.pdf
```

## Library

```go
import "github.com/Marist-Community-and-Belonging/report-gen-pdf/reportgen"

report, err := reportgen.BuildFromProcessed(processedDir, 2023)
pdf, err := reportgen.Generate(report, reportgen.Options{
  OutPath:  "out/report.pdf",
  TypstDir: "out/report-typst",
})
```

`Generate` returns PDF bytes and is safe for concurrent use. `BuildFromProcessed`
reads `data/processed/*.json` only; it never fetches IPEDS.
Use `BuildFromFS` when the extracts are embedded in an application binary.

Compile an exported Typst project from its directory:

```bash
typst compile --font-path assets/fonts templates/report.typ report.pdf
```

## Preview templates in-repo

`reportgen/templates/report.typ` defaults to `../data.json`. That path is a
symlink to `examples/sample.json` so the Typst language server and:

```bash
cd reportgen
typst compile --font-path assets/fonts --root . templates/report.typ /tmp/preview.pdf
```

work without a prior export. `Generate` always passes `--input data=…` and
ignores the default.

## JSON shape

| Field | Notes |
|---|---|
| `chapters[]` | `heading`, `body`; optional `bullets`, `figures`, `callouts` |
| `figures[].kind` | `bar` \| `grouped` \| `line` \| `bullet` \| `bullet-grid` \| `stacked` \| `scatter` \| `stats` |
| `unit` | `percent` \| `usd` \| `pp` \| `index` \| `count` |
| optional | `intro`, `release`, `notes`, `office`, `phone`, `author` |

See `examples/sample.json`.

## Chapters (`BuildFromProcessed`)

| Chapter | Dashboard page |
|---|---|
| Overview | `index.html` |
| Marist profile | `marist-profile.html` |
| Multi-year trends | `index.html` trends |
| Diversity and access | `diversity-access.html` |
| Success and equity | `success-equity.html` |
| Affordability | `affordability-resources.html` |

Campus Involvement (live CampusGroups feed) is omitted. Empty-figure chapters
are dropped (e.g. no trends chapter for the earliest collection).

## Layout

```
cmd/reportgen/           CLI
examples/sample.json     hand-authored demo
reportgen/
  data.json              → ../examples/sample.json (IDE / local Typst)
  generate.go            Typst compile + asset unpack
  dashboard.go           BuildFromProcessed
  templates/             report.typ, plots.typ
  assets/fonts/          Inter + Source Serif Pro (SIL OFL 1.1)
  assets/ocb-header.jpg  letterhead
out/                     generated PDFs (gitignored)
```
