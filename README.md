# pdfa11y

`pdfa11y` is an open-source command-line PDF/UA accessibility checker
written in Go. It runs Matterhorn-protocol-style checks against PDF
documents and emits either a human-readable report card or structured
JSON, so it can drive both interactive review and batch / CI gates.

It targets the gap left by [PAC](https://pac.pdf-accessibility.org/en),
which is the de-facto standard for PDF/UA verification but is GUI-only.

A browser-based version of the same checker runs at
[pdfuacheck.speedata.de](https://pdfuacheck.speedata.de). The check runs
locally in the browser (WebAssembly); the PDF never leaves the machine.

## Status

19 checks across 9 Matterhorn categories (structure tree, metadata,
viewer preferences, fonts, natural language, graphics, headings,
tables, lists). Structure-tree walking with role-map resolution, font
enumeration, XMP introspection, content-stream tokenisation for
used-fonts and MCID-consistency checks, and tri-state reporting are
in place. Validated against the [pdfa.org technique sample corpus](https://pdfa.org/techniques-for-accessible-pdf/)
(82 reference PDFs); current match rate is around 62%, with 0 known
false positives -- divergences from PAC are all stricter findings or
Matterhorn human checks that need a person in the loop.

Still missing: annotation / form-field checks, font-glyph-level
encoding analysis, reading-order heuristics. See [Roadmap](#roadmap).

## Install

Pre-built binaries for Linux, macOS and Windows are on the
[GitHub releases page](https://github.com/speedata/pdfa11y/releases).
Each release contains tar.gz archives for the Unix platforms and a
zip for Windows; macOS binaries are signed and notarized.

From source:

```sh
go install github.com/speedata/pdfa11y/cmd/pdfa11y@latest
```

Requires Go 1.23 or later.

## Usage

```sh
# Single document, human-readable report
pdfa11y document.pdf

# Batch run, one JSON object per file (stream-friendly)
pdfa11y --format jsonl *.pdf > results.jsonl

# Pretty JSON array of all documents (one parse, ideal for tooling)
pdfa11y --format json a.pdf b.pdf | jq '.[] | {path, verdict}'

# Show WCAG mapping next to each check; treat warnings as errors
pdfa11y --wcag --strict report.pdf

# List every registered check and exit
pdfa11y --list-rules
```

### Exit codes

| Code | Meaning |
| --- | --- |
| 0 | All documents pass (no error-severity findings) |
| 1 | At least one document fails |
| 2 | Tool error or invalid usage |

With `--strict`, warning-severity findings count as errors and feed
into the exit code accordingly.

## Implemented checks

All checks currently apply to both PDF/UA-1 and PDF/UA-2. With
`--spec auto` (default) the spec is detected per document via
`pdfuaid:part` in the XMP metadata; `--spec pdfua1` or `pdfua2`
forces a specific set.

| ID | Category | Title | WCAG |
| --- | --- | --- | --- |
| MH-01-002 | Structure tree | MarkInfo declares the document as marked | 1.3.1 |
| MH-01-005 | Structure tree | Document has a structure tree | 1.3.1 |
| MH-02-003 | Structure tree | Content-stream MCIDs match the structure tree | 1.3.1 |
| MH-06-001 | Metadata | Document has a title in metadata | 2.4.2 |
| MH-06-003 | Metadata | XMP metadata declares PDF/UA identifier | — |
| MH-06-004 | Metadata | XMP metadata contains dc:title | 2.4.2 |
| MH-07-001 | Viewer preferences | ViewerPreferences/DisplayDocTitle is true | 2.4.2 |
| MH-09-001 | Fonts | All fonts are embedded | — |
| MH-10-001 | Fonts | All fonts have a /ToUnicode map | 1.3.1 |
| MH-11-001 | Natural language | Document declares a primary language | 3.1.1 |
| MH-13-004 | Graphics | Figure has Alt or ActualText | 1.1.1 |
| MH-14-001 | Structure tree | Real content is inside a marked-content sequence or Artifact | 1.3.1 |
| MH-14-003 | Headings | Headings start at H1 and are properly nested | 1.3.1, 2.4.6 |
| MH-14-006 | Headings | Heading style is consistent (H or H&lt;n&gt;, not both) | 1.3.1, 2.4.6 |
| MH-15-003 | Tables | Table contains rows (TR) | 1.3.1 |
| MH-16-001 | Lists | List contains list items (LI) | 1.3.1 |
| MH-16-002 | Lists | List items contain LBody | 1.3.1 |
| MH-16-003 | Lists | Lists declare /ListNumbering (Warning) | 1.3.1 |
| MH-31-008 | Structure tree | Custom structure types are mapped to standard types | 1.3.1 |

Font checks (MH-09-001, MH-10-001) only flag fonts that are actually
referenced from a content stream, not fonts declared in /Resources
and never used. MH-16-003 is severity Warning rather than Error: the
attribute defaults to None and is legitimately omitted on unordered
lists, and pdfa11y cannot distinguish ordered from unordered without
inspecting the Lbl glyph.

Run `pdfa11y --list-rules` for the same list at runtime.

## Output formats

- `terminal` (default): tri-state PASS / WARN / FAIL per check,
  grouped by Matterhorn category, with hints for each finding.
- `json`: a single top-level array of document objects, indented.
  Each document carries `path`, `verdict`, `summary` and `results`.
- `jsonl`: JSON Lines, one compact document per line.
- `html`: standalone HTML report card with the same content as the
  terminal output, colour-coded and printable.

The JSON schema is stable enough to consume from Go programs (the engine
package exposes `MarshalJSON`/`UnmarshalJSON` for `Verdict`, `Spec` and
`Severity`); ad-hoc consumers can also rely on the documented field
names being kept backwards-compatible.

## Architecture

```
cmd/pdfa11y/        CLI (optionparser)
cmd/pdfa11ywasm/    WebAssembly entry point (drives pdfuacheck.speedata.de)
cmd/genfixtures/    Fixture regenerator
internal/engine/    Check interface, registry, runner, Verdict
internal/model/     Document/Dict/StructElement/Font/PageReport interfaces
internal/pdf/       pdfdisassembler-backed implementation of the model,
                    incl. per-page content-stream walker
internal/pdfua/     Shared helpers for the PDF/UA XMP identifier
internal/checks/    Individual checks, one Matterhorn category per package
internal/report/    Output formatters (terminal, json, html)
internal/realworld/ Cross-validation harness against the pdfa.org corpus
web/                WASM frontend (HTML + JS + wasm_exec.js + pdfa11y.wasm)
```

The parsing layer lives in a separate library,
[pdfdisassembler](https://github.com/speedata/pdfdisassembler), so the
PDF-reading code can be reused by other tooling. Checks depend only
on the `model` interface, so the backend can be swapped without
touching them. New checks plug in via `engine.Register(checkInstance)`
in an `init()` and are picked up by the CLI without touching engine
or main.

## Development

```sh
# Run all tests
go test ./...

# Regenerate fixtures after changing the generator
go run ./cmd/genfixtures
```

Fixtures live alongside their tests in `internal/checks/*/testdata/`
and are checked in. They are derived from one canonical base PDF.

### Cross-validation against an external reference corpus

`internal/realworld` carries a `TestReferenceCorpus` test that walks
the [pdfa.org technique sample PDFs](https://pdfa.org/techniques-for-accessible-pdf/)
and tabulates pdfa11y's verdict against each file's filename-encoded
expectation (`_F<n>` means "should fail", anything else means
"should pass"). The test never fails on a mismatch — it produces a
`CROSS_VALIDATION.md` report at the repo root for human review.

```sh
# Without PDFA11Y_REFCORPUS the test skips silently:
go test ./internal/realworld/

# Set the env var to a corpus checkout to run it:
PDFA11Y_REFCORPUS=/path/to/techniques-for-accessible-pdf \
  go test ./internal/realworld/ -run TestReferenceCorpus -v
```

`CROSS_VALIDATION.md` is gitignored — re-run the test to regenerate
it. Per-file expectations (orthogonal-but-real findings on PASS
samples) are recorded in `internal/realworld/refcorpus_expectations_test.go`.

The reference PDFs are © pdfa.org and licensed under
[CC-BY-4.0](https://creativecommons.org/licenses/by/4.0/). pdfa.org
asks that consumers re-fetch the files directly from the technique
pages rather than caching local snapshots, as the samples can change
during the standard's development phase. The pdfa11y repository
therefore does not vendor them.

## Roadmap

Short-term:
- Annotation walker so link / form / widget checks can land (MH-28
  family: links have `/Contents`, form fields have `/TU` tooltips)
- Cell-level table rules: TR contains only TH/TD; TH declares /Scope
- `/Lang` per structure element (companion to MH-11-001)
- Outlines required on documents above a length threshold
- Parallel batch processing (`--jobs N`)
- GitHub Actions CI on PRs

Longer-term:
- Pragmatic glyph-analysis subset: parse the TrueType cmap of
  embedded fonts to distinguish mis-declared symbolic fonts from real
  Latin fonts, closing the F03-class divergence with PAC
- Reading-order heuristics (MH-09 / G4 family) once annotation
  walking is in place
- WCAG-only filtering for documents that are not formally PDF/UA but
  should still satisfy WCAG-equivalent accessibility expectations

## License

MIT — see [`LICENSE`](LICENSE). Third-party dependency licenses are
enumerated in [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md).
