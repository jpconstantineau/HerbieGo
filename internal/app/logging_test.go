package app

import (
	"io"
	"os"
	"testing"
)

func TestNewDiscardLoggerDropsOutput(t *testing.T) {
	originalStderr := os.Stderr
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = writePipe
	t.Cleanup(func() {
		os.Stderr = originalStderr
		_ = writePipe.Close()
		_ = readPipe.Close()
	})

	logger := NewDiscardLogger()
	logger.Info("this should not be written")
	if err := writePipe.Close(); err != nil {
		t.Fatalf("writePipe.Close() error = %v", err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if len(output) != 0 {
		t.Fatalf("captured stderr = %q, want no output", string(output))
	}
}
