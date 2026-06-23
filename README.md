[![Explore in Constellation](https://img.shields.io/badge/Explore%20in-Constellation-blue)](https://constellation.speedata.de)


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
fonts, MCID consistency, untagged content, per-font code coverage,
MCID bounding boxes for reading-order heuristics), annotation
walking, and tri-state reporting are in place. Cross-validated
against the [pdfa.org technique sample corpus](https://pdfa.org/techniques-for-accessible-pdf/)
(82 reference PDFs); divergences from PAC are typically stricter
findings on pdfa11y's side or Matterhorn conditions that need a
human reviewer. See [Development](#development) to reproduce the
cross-validation locally.

Run `pdfa11y --list-rules` for the current, build-specific list of registered checks.

Still missing: font-glyph-level encoding analysis, finer-grained
reading-order checks (within-tag MCID order, sidebar placement),
parallel batch processing.

## How it compares

PAC and veraPDF are the established PDF/UA checkers, and pdfa11y does
not try to replace them for formal validation. veraPDF is the
ISO-backed reference validator for the complete PDF/UA rule set; PAC
is the desktop tool most accessibility practitioners reach for.
pdfa11y deliberately implements the machine-checkable subset of the
Matterhorn conditions, and optimises for the cases the other two
leave awkward:

- **Scriptable and CI-native.** A single static Go binary. No JVM,
  no GUI. Meaningful exit codes, JSON Lines for streaming, and a
  stable JSON schema for tooling. PAC is a Windows desktop
  application; veraPDF needs a Java runtime.
- **Cross-platform.** Native Linux, macOS and Windows binaries (the
  macOS build is signed and notarized). PAC is Windows-only.
- **Actionable findings.** Every finding carries a plain-language
  hint plus a WCAG mapping and the spec clause it derives from, so
  the report tells an author what to change — not just that something
  is wrong. veraPDF emits a machine compliance report; PAC reports
  pass/fail per check.
- **PDF/UA-2 aware.** The spec is auto-detected per document from
  `pdfuaid:part`, and the UA-1 vs UA-2 differences (heading model,
  `FENote`, the PDF 2.0 namespace, XFA removal) are modelled rather
  than retrofitted.
- **Self-checking output.** The optional PDF report is itself a
  tagged PDF/UA-1 document and passes pdfa11y's own checks.
- **Permissive license.** MIT, so it can be embedded in commercial
  pipelines; veraPDF is GPL/MPL, PAC is proprietary.

| | pdfa11y | PAC | veraPDF |
| --- | --- | --- | --- |
| Interface | CLI + JSON/JSONL | desktop GUI | CLI / Java API / REST |
| Platforms | Linux, macOS, Windows | Windows only | any with a JVM |
| CI-native | yes (exit codes, JSONL) | no | yes |
| Per-finding fix hints | yes | some | spec clause only |
| Rule coverage | machine-checkable subset | comprehensive | comprehensive (ISO reference) |
| License | MIT | proprietary | GPL/MPL |

For an exhaustive, audit-grade PDF/UA verdict, veraPDF remains the
reference. pdfa11y is the lightweight, embeddable, developer- and
author-friendly checker that sits next to it — and is cross-validated
against the same pdfa.org corpus, where the remaining divergences are
stricter findings on pdfa11y's side or Matterhorn conditions that
need a human reviewer.

## Install

Pre-built binaries for Linux, macOS and Windows are on the
[GitHub releases page](https://github.com/speedata/pdfa11y/releases).
Each release contains tar.gz archives for the Unix platforms and a
zip for Windows; macOS binaries are signed and notarized.

From source:

```sh
go install github.com/speedata/pdfa11y/cmd/pdfa11y@latest
```

Requires Go 1.25 or later.

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
| 06 Metadata | XMP `dc:title`, `pdfuaid:part`, `pdfuaid:rev`, DocInfo/XMP title agreement |
| 07 Viewer preferences | DisplayDocTitle |
| 08 Tab order | `/Tabs` = S |
| 09 Fonts | Embedding, `/ToUnicode` presence + coverage, CIDFontType2 `/CIDToGIDMap` |
| 11 Natural language | Catalog `/Lang`, per-element `/Lang` coverage |
| 12 Embedded files | Associated Files declare `/AFRelationship` |
| 13 Graphics | Figure Alt / ActualText |
| 14 Headings | Hierarchy, mixed H / Hn styles |
| 15 Tables | Rows, TR child shape, TH `/Scope` |
| 16 Lists | LI presence, LI / LBody, `/ListNumbering` |
| 17 Math | Formula Alt / ActualText |
| 19 Notes and references | Note `/ID` present, references resolve |
| 20 Optional content | OCG `/Name` |
| 26 Security | Encryption permits AT extraction |
| 27 Navigation | Outlines on multi-page documents |
| 28 Annotations and forms | Link `/Contents`, form `/TU`, struct-tree linkage, artifact subtypes, off-page hiding, AcroForm fields in structure tree |

All checks apply to both PDF/UA-1 and PDF/UA-2. `--spec auto`
(default) detects the spec per document via `pdfuaid:part` in the
XMP metadata; `--spec pdfua1` / `pdfua2` forces a specific set.

A few checks have severity Warning rather than Error where the
spec leaves room (e.g. UA-16-003 `/ListNumbering` defaults to
None on unordered lists; UA-27-001 outlines on documents above a
conventional length threshold). Font checks (UA-09-001, UA-10-001)
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
- `pdf`: PDF/UA-1 accessibility report (title page + per-document
  findings, structurally tagged) rendered via boxesandglue/bagme.
  The output PDF itself passes pdfa11y's own checks.

The JSON schema is stable enough to consume from Go programs (the engine
package exposes `MarshalJSON`/`UnmarshalJSON` for `Verdict`, `Spec` and
`Severity`); ad-hoc consumers can also rely on the documented field
names being kept backwards-compatible.

## Architecture

```
cmd/pdfa11y/        CLI (optionparser)
cmd/inspect/        Debug helper (DocInfo, fonts, struct-tree shape)
cmd/genfixtures/    Fixture regenerator
internal/engine/    Check interface, registry, runner, Verdict
internal/model/     Document/Dict/StructElement/Font/PageReport interfaces
internal/pdf/       pdfdisassembler-backed implementation of the model,
                    incl. per-page content-stream walker
internal/pdfua/     Shared helpers for the PDF/UA XMP identifier
internal/checks/    Individual checks, one Matterhorn category per package
internal/report/    Output formatters (terminal, json, html, pdf)
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
"should pass"). The test never fails on a mismatch, it produces a
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

## Ecosystem

pdfa11y is part of a broader ecosystem of PDF, typesetting and publishing technologies.

**[Explore the constellation →](https://constellation.speedata.de)**

## License

MIT — see [`LICENSE`](LICENSE). Third-party dependency licenses are
enumerated in [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md).
