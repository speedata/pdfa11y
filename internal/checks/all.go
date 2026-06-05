// Package checks blank-imports every check package so that init()-time
// registration happens by importing this single package.
package checks

import (
	_ "github.com/speedata/pdfa11y/internal/checks/fonts"
	_ "github.com/speedata/pdfa11y/internal/checks/graphics"
	_ "github.com/speedata/pdfa11y/internal/checks/headings"
	_ "github.com/speedata/pdfa11y/internal/checks/language"
	_ "github.com/speedata/pdfa11y/internal/checks/lists"
	_ "github.com/speedata/pdfa11y/internal/checks/metadata"
	_ "github.com/speedata/pdfa11y/internal/checks/navigation"
	_ "github.com/speedata/pdfa11y/internal/checks/structure"
	_ "github.com/speedata/pdfa11y/internal/checks/tables"
	_ "github.com/speedata/pdfa11y/internal/checks/viewerprefs"
)
