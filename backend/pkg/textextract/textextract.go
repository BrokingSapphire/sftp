// Package textextract pulls best-effort plain text out of common document
// formats for full-text indexing. It is dependency-light: plain/code files are
// read directly, PDFs via a pure-Go reader, and Office (docx/xlsx/pptx) by
// unzipping and stripping their XML. Unsupported types return empty text (not
// an error) so the caller can index everything uniformly.
package textextract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// MaxTextBytes caps how much text we keep per file (bounds tsvector size and
// memory). ~1 MiB of text is far more than any keyword search needs.
const MaxTextBytes = 1 << 20

// MaxInputBytes caps how much of a file we will read into memory to parse.
const MaxInputBytes = 64 << 20 // 64 MiB

// Supported reports whether Extract can produce text for this type.
func Supported(ext, mime string) bool {
	return kindOf(ext, mime) != kindNone
}

type kind int

const (
	kindNone kind = iota
	kindPlain
	kindPDF
	kindOOXMLDoc // docx, pptx (paragraph-style XML)
	kindXLSX
)

var plainExts = map[string]bool{
	"txt": true, "md": true, "markdown": true, "csv": true, "tsv": true, "log": true,
	"json": true, "yaml": true, "yml": true, "xml": true, "html": true, "htm": true,
	"go": true, "py": true, "js": true, "ts": true, "tsx": true, "jsx": true, "java": true,
	"c": true, "h": true, "cpp": true, "cc": true, "cs": true, "rb": true, "php": true,
	"rs": true, "sh": true, "sql": true, "toml": true, "ini": true, "env": true, "conf": true,
}

func kindOf(ext, mime string) kind {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "pdf":
		return kindPDF
	case "docx", "pptx":
		return kindOOXMLDoc
	case "xlsx":
		return kindXLSX
	}
	if plainExts[ext] {
		return kindPlain
	}
	if strings.HasPrefix(mime, "text/") || mime == "application/json" || mime == "application/xml" {
		return kindPlain
	}
	return kindNone
}

// Extract returns best-effort plain text. ext is the file extension (no dot),
// mime the detected content type. Never panics; errors are returned, not raised.
func Extract(ext, mime string, r io.Reader) (text string, err error) {
	defer func() {
		// Some third-party parsers panic on malformed input — contain it.
		if rec := recover(); rec != nil {
			text, err = "", nil
		}
	}()

	switch kindOf(ext, mime) {
	case kindPlain:
		return readPlain(r)
	case kindPDF:
		return extractPDF(r)
	case kindOOXMLDoc:
		return extractOOXML(r, "")
	case kindXLSX:
		return extractOOXML(r, "xl/")
	default:
		return "", nil
	}
}

func readPlain(r io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxTextBytes))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// extractOOXML unzips an Office file and concatenates the character data of the
// relevant XML parts. prefix filters entries (e.g. "xl/" for xlsx shared
// strings); empty prefix scans document/slide parts.
func extractOOXML(r io.Reader, prefix string) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxInputBytes))
	if err != nil {
		return "", err
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, f := range zr.File {
		name := f.Name
		if !relevantPart(name, prefix) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		writeXMLText(&out, rc)
		rc.Close()
		if out.Len() >= MaxTextBytes {
			break
		}
	}
	return truncate(out.String()), nil
}

func relevantPart(name, prefix string) bool {
	if prefix == "xl/" {
		// Spreadsheet text lives in shared strings and inline sheet strings.
		return name == "xl/sharedStrings.xml" || strings.HasPrefix(name, "xl/worksheets/sheet")
	}
	// Word + PowerPoint body text.
	return name == "word/document.xml" ||
		(strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml")) ||
		strings.HasPrefix(name, "word/header") || strings.HasPrefix(name, "word/footer")
}

// writeXMLText streams an XML document and appends its character data, adding
// spaces between elements so words don't run together.
func writeXMLText(out *strings.Builder, r io.Reader) {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		if cd, ok := tok.(xml.CharData); ok {
			s := strings.TrimSpace(string(cd))
			if s != "" {
				out.WriteString(s)
				out.WriteByte(' ')
			}
			if out.Len() >= MaxTextBytes {
				return
			}
		}
	}
}

func truncate(s string) string {
	if len(s) > MaxTextBytes {
		return s[:MaxTextBytes]
	}
	return s
}
