package xlog

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotatingOutputPathBuildsSink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xbase.log")
	paths := rotatingOutputPaths([]string{path}, rotationConfig{
		Enabled:   true,
		Daily:     true,
		MaxSize:   1,
		LocalTime: true,
	})
	if len(paths) != 1 || paths[0] == path {
		t.Fatalf("rotatingOutputPaths() = %v", paths)
	}

	target, err := url.Parse(paths[0])
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	sink, err := newRotatingSink(target)
	if err != nil {
		t.Fatalf("newRotatingSink() error = %v", err)
	}
	if _, err = sink.Write([]byte("complete entry\n")); err != nil {
		t.Fatalf("sink.Write() error = %v", err)
	}
	if err = sink.Close(); err != nil {
		t.Fatalf("sink.Close() error = %v", err)
	}
	if content, readErr := os.ReadFile(path); readErr != nil || !bytes.Equal(content, []byte("complete entry\n")) {
		t.Fatalf("rotating sink output = %q, err = %v", content, readErr)
	}
}

func TestRotatingSinkPreservesWholeWritesAtSizeBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xbase.log")
	sink := newRotatingFileSink(path, rotationConfig{MaxSize: 1})

	first := append([]byte("first:"), bytes.Repeat([]byte("a"), 600*1024)...)
	first = append(first, '\n')
	second := append([]byte("second:"), bytes.Repeat([]byte("b"), 600*1024)...)
	second = append(second, '\n')
	writeAndClose(t, sink, first, second)

	assertWritesPreserved(t, dir, first, second)
}

func TestRotatingSinkPreservesWholeWritesAtDayBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xbase.log")
	sink := newRotatingFileSink(path, rotationConfig{Daily: true, MaxSize: 100, LocalTime: true})

	now := time.Date(2026, time.July, 25, 23, 59, 59, 0, time.Local)
	sink.now = func() time.Time { return now }
	sink.day = sink.currentDay()
	first := []byte("before midnight\n")
	if _, err := sink.Write(first); err != nil {
		t.Fatalf("write before day boundary: %v", err)
	}

	now = now.Add(2 * time.Second)
	second := []byte("after midnight\n")
	writeAndClose(t, sink, second)

	assertWritesPreserved(t, dir, first, second)
}

func writeAndClose(t *testing.T, sink *rotatingSink, writes ...[]byte) {
	t.Helper()
	for _, data := range writes {
		written, err := sink.Write(data)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if written != len(data) {
			t.Fatalf("Write() wrote %d bytes, want %d", written, len(data))
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func assertWritesPreserved(t *testing.T, dir string, writes ...[]byte) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "xbase*.log"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("rotation created %d files, want at least 2", len(files))
	}

	var combined []byte
	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil {
			t.Fatalf("ReadFile(%q) error = %v", file, readErr)
		}
		combined = append(combined, content...)
	}
	for _, data := range writes {
		if count := bytes.Count(combined, data); count != 1 {
			t.Fatalf("complete write appears %d times, want exactly once", count)
		}
	}
}
