# Third-party licenses

pdfa11y is distributed under the MIT License (see `LICENSE`). The compiled
binary statically links the third-party Go modules listed below; their
licenses apply to the corresponding portions of the binary. Versions
reflect the current `go.mod`; update this file when dependencies change.

| Module | Version | License |
| --- | --- | --- |
| github.com/pdfcpu/pdfcpu | v0.12.1 | Apache-2.0 |
| github.com/speedata/optionparser | v1.2.1 | MIT |
| github.com/hhrutter/tiff | v1.0.3 | BSD-3-Clause |
| github.com/hhrutter/lzw | v1.0.0 | BSD-3-Clause |
| github.com/hhrutter/pkcs7 | v0.2.2 | MIT |
| github.com/mattn/go-runewidth | v0.0.23 | MIT |
| github.com/clipperhouse/uax29/v2 | v2.7.0 | MIT |
| github.com/pkg/errors | v0.9.1 | BSD-2-Clause |
| golang.org/x/crypto | v0.50.0 | BSD-3-Clause |
| golang.org/x/image | v0.39.0 | BSD-3-Clause |
| golang.org/x/text | v0.36.0 | BSD-3-Clause |
| gopkg.in/yaml.v2 | v2.4.0 | Apache-2.0 |
| gopkg.in/yaml.v3 | v3.0.1 | Apache-2.0, MIT |

The full license texts ship with each module in the Go module cache and
are reproduced in upstream repositories. The Apache-2.0 NOTICE that
accompanies `gopkg.in/yaml.v2` is reproduced below; pdfcpu has no NOTICE
file of its own.

## NOTICE — gopkg.in/yaml.v2

```
Copyright 2011-2016 Canonical Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## veraPDF validation profiles (CC BY 4.0)

The veraPDF coverage manifest (`internal/verasync/coverage.yaml`) reproduces
rule descriptions, ISO clause references, and test identifiers from the
**veraPDF validation profiles**, used here as a coverage map and test oracle
(see `VERAPDF_SYNC_PLAN.md`). The profiles are not linked into the pdfa11y
binary; only their textual rule metadata is embedded in the manifest.

- Source: <https://github.com/veraPDF/veraPDF-validation-profiles>
- Copyright: © veraPDF Consortium
- License: Creative Commons Attribution 4.0 International (CC BY 4.0),
  <https://creativecommons.org/licenses/by/4.0/>
- Pinned revision: see `internal/verasync/PINNED_VERSION`

The requirement wording ultimately derives from ISO 14289-1:2014 and
ISO 14289-2:2024; pdfa11y treats the ISO clause as the primary source and the
veraPDF rule ID as a cross-reference.
