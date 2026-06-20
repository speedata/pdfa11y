package realworld_test

// allowedFailures lists checks whose failure on a specific pdfa.org
// reference file is known to be a legitimate finding that is
// orthogonal to the technique that file is meant to demonstrate.
//
// pdfa.org's "_NN" PASS files are not full-conformance fixtures: they
// are focused samples that illustrate one technique correctly. They
// can carry other unrelated PDF/UA defects that pdfa11y rightly flags
// but that would not count as a "false positive" in the
// cross-validation sense -- the demonstrated technique is correct;
// some other detail isn't.
//
// Key: the path relative to the corpus root (e.g.
// "fundamentals/2-text/G2_06-.../UA1_Tpdf-G2_06.pdf").
// Value: UA-IDs of checks whose failure is documented as expected on
// that file and therefore not counted toward the false-positive total.
//
// Add an entry only after manually inspecting the file and confirming
// the finding is real-but-orthogonal. Each entry should be justified
// with a comment.
var allowedFailures = map[string][]string{
	// G2_06 demonstrates /ActualText correcting OCR-mis-recognised
	// glyphs. The page additionally carries a `BDC /Figure /MCID 2`
	// marker in its content stream whose MCID is not referenced from
	// the structure tree (ParentTree slot 2 is empty). That is a real
	// PDF/UA-1 §7.1 / ISO 32000-1 §14.7.4.3 violation, but it is
	// orthogonal to the ActualText technique this sample is showing.
	// PAC/pdfa.org list the file as a PASS example of the technique;
	// pdfa11y's UA-02-003 finding is correct on its own terms.
	"fundamentals/2-text/G2_06-ActualText-provides-correct-extractable-characters-in-place-of-OCR-errors/UA1_Tpdf-G2_06.pdf": {"UA-02-003"},
}
