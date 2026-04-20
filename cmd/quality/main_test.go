package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileNeedsFormattingIgnoresLineEndingOnlyDifferences(t *testing.T) {
	path := writeGoFile(t, "sample.go", "package sample\r\n\r\nfunc answer() int {\r\n\treturn 42\r\n}\r\n")

	needsFormatting, err := fileNeedsFormatting(path)
	if err != nil {
		t.Fatalf("fileNeedsFormatting() error = %v", err)
	}
	if needsFormatting {
		t.Fatal("fileNeedsFormatting() = true, want false for CRLF-only differences")
	}
}

func TestFileNeedsFormattingDetectsRealFormattingDifferences(t *testing.T) {
	path := writeGoFile(t, "sample.go", "package sample\n\nfunc answer() int {\n    return 42\n}\n")

	needsFormatting, err := fileNeedsFormatting(path)
	if err != nil {
		t.Fatalf("fileNeedsFormatting() error = %v", err)
	}
	if !needsFormatting {
		t.Fatal("fileNeedsFormatting() = false, want true for unformatted Go source")
	}
}

func writeGoFile(t *testing.T, name string, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
