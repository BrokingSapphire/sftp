package textextract

import (
	"bytes"
	"io"

	"github.com/ledongthuc/pdf"
)

// extractPDF pulls plain text from a PDF using a pure-Go reader (no external
// binaries). Encrypted or image-only PDFs yield little/no text — OCR handles
// those in a later pass.
func extractPDF(r io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxInputBytes))
	if err != nil {
		return "", err
	}
	pr, err := pdf.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", err
	}
	tr, err := pr.GetPlainText()
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(io.LimitReader(tr, MaxTextBytes))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
