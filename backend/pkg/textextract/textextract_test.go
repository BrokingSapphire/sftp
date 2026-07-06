package textextract

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestPlain(t *testing.T) {
	got, err := Extract("txt", "text/plain", strings.NewReader("hello world foo"))
	if err != nil || !strings.Contains(got, "hello world") {
		t.Fatalf("plain: %q err %v", got, err)
	}
}

func TestCodeAndJSON(t *testing.T) {
	if !Supported("go", "") || !Supported("json", "application/json") {
		t.Fatal("code/json should be supported")
	}
	got, _ := Extract("json", "application/json", strings.NewReader(`{"key":"value123"}`))
	if !strings.Contains(got, "value123") {
		t.Fatalf("json extract: %q", got)
	}
}

func TestUnsupportedReturnsEmpty(t *testing.T) {
	got, err := Extract("png", "image/png", strings.NewReader("\x89PNG..."))
	if err != nil || got != "" {
		t.Fatalf("unsupported should be empty/no-error: %q %v", got, err)
	}
	if Supported("png", "image/png") {
		t.Fatal("png should not be supported")
	}
}

func TestDocx(t *testing.T) {
	// Build a minimal .docx (zip with word/document.xml).
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/document.xml")
	w.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="x"><w:body><w:p><w:r><w:t>Quarterly</w:t></w:r><w:r><w:t>Report</w:t></w:r></w:p></w:body></w:document>`))
	zw.Close()

	got, err := Extract("docx", "", bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Quarterly") || !strings.Contains(got, "Report") {
		t.Fatalf("docx extract missing words: %q", got)
	}
}

func TestXlsx(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("xl/sharedStrings.xml")
	w.Write([]byte(`<sst><si><t>Invoice</t></si><si><t>Acme</t></si></sst>`))
	zw.Close()

	got, _ := Extract("xlsx", "", bytes.NewReader(buf.Bytes()))
	if !strings.Contains(got, "Invoice") || !strings.Contains(got, "Acme") {
		t.Fatalf("xlsx extract: %q", got)
	}
}
