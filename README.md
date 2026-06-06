# pdfa11y

`pdfa11y` is an open-source command-line PDF/UA accessibility checker
written in Go. It runs Matterhorn-protocol-style checks against PDF
documents and emits either a human-readable report card or structured
JSON, so it can drive both interactive review and batch / CI gates.

It targets the gap left by [PAC](https://pac.pdf-accessibility.org/en),
which is the de-facto standard for PDF/UA verification but is GUI-only.


## Status

Several dozen checks across the Matterhorn schedules listed under
[Implemented checks](#implemented-checks). Structure-tree walking
with role-map resolution, font enumeration with /ToUnicode CMap
parsing, XMP introspection, content-stream tokenisation (used
fonts, MCID consistency, untagged content, per-font code coverage),
annotation walking, and tri-state reporting are in place. Validated
against the [pdfa.org technique sample corpus](https://pdfa.org/techniques-for-accessible-pdf/)
(82 reference PDFs); current match rate is around 66%, with 0 known
false positives -- divergences from PAC are stricter findings or
Matterhorn human checks that need a person in the loop.

Run `pdfa11y --list-rules` for the current, build-specific list of registered checks.

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

`pdfa11y --list-rules` prints the authoritative, version-specific
list with IDs, titles, severities and WCAG mappings.

The check set currently spans these Matterhorn categories:

| Schedule | Coverage |
| --- | --- |
| 01 Real content / Structure tree | MarkInfo, StructTreeRoot, MCID consistency, untagged content, custom-tag role-map |
| 06 Metadata | DocInfo title, XMP `dc:title`, `pdfuaid:part`, title agreement |
| 07 Viewer preferences | DisplayDocTitle |
| 08 Tab order | `/Tabs` = S |
| 09 Fonts | Embedding, `/ToUnicode` presence, `/ToUnicode` coverage |
| 11 Natural language | Catalog `/Lang` |
| 13 Graphics | Figure Alt / ActualText |
| 14 Headings | Hierarchy, mixed H / Hn styles |
| 15 Tables | Rows, TR child shape, TH `/Scope` |
| 16 Lists | LI presence, LI / LBody, `/ListNumbering` |
| 17 Math | Formula Alt / ActualText |
| 20 Optional content | OCG `/Name` |
| 26 Security | Encryption permits AT extraction |
| 27 Navigation | Outlines on multi-page documents |
| 28 Annotations and forms | Link `/Contents`, form `/TU`, struct-tree linkage, artifact subtypes, off-page hiding |

All checks apply to both PDF/UA-1 and PDF/UA-2. `--spec auto`
(default) detects the spec per document via `pdfuaid:part` in the
XMP metadata; `--spec pdfua1` / `pdfua2` forces a specific set.

A few checks have severity Warning rather than Error where the
spec leaves room (e.g. MH-16-003 `/ListNumbering` defaults to
None on unordered lists; MH-27-001 outlines on documents above a
conventional length threshold). Font checks (MH-09-001, MH-10-001)
only flag fonts that are actually referenced from a content
stream, not fonts declared in `/Resources` and never used.

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
cmd/genfixtures/    Fixture regenerator
internal/engine/    Check interface, registry, runner, Verdict
internal/model/     Document/Dict/StructElement/Font/PageReport interfaces
internal/pdf/       pdfdisassembler-backed implementation of the model,
                    incl. per-page content-stream walker
internal/pdfua/     Shared helpers for the PDF/UA XMP identifier
internal/checks/    Individual checks, one Matterhorn category per package
internal/report/    Output formatters (terminal, json, html)
internal/realworld/ Cross-validation harness against the pdfa.org corpus
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
