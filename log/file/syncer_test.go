package file

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xbaseio/xbase/log/internal"
)

func TestMakeFileName(t *testing.T) {
	s := NewSyncer(
		WithPath(filepath.Join(t.TempDir(), "xbase.log")),
		WithRotate(RotateDay),
	)
	defer func() { _ = s.Close() }()

	if got := s.makeFileName("20260710", 0, ".log"); got != "xbase.20260710.log" {
		t.Fatalf("unexpected current file name: %s", got)
	}

	if got := s.makeFileName("20260710", 2, ".log"); got != "xbase.20260710.2.log" {
		t.Fatalf("unexpected rotated file name: %s", got)
	}
}

func TestSyncerCurrentFileUsesDateTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xbase.log")
	s := NewSyncer(
		WithPath(path),
		WithRotate(RotateDay),
		WithMaxSize(1024*1024),
	)
	defer func() { _ = s.Close() }()

	now := time.Now()
	if err := s.Write(&internal.Entity{
		Now:     now,
		Time:    now.Format(time.DateTime),
		Level:   internal.LevelInfo,
		Message: "hello",
	}); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	want := filepath.Join(dir, "xbase."+now.Format("20060102")+".log")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected current dated file %s: %v", want, err)
	}
}

func TestSyncerRotateBySizeUsesVersionInSameDay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xbase.log")
	s := NewSyncer(
		WithPath(path),
		WithRotate(RotateDay),
		WithMaxSize(32),
	)
	defer func() { _ = s.Close() }()

	now := time.Now()
	write := func(msg string) {
		if err := s.Write(&internal.Entity{
			Now:     now,
			Time:    now.Format(time.DateTime),
			Level:   internal.LevelInfo,
			Message: msg,
		}); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	write("abcdefghijklmnopqrstuvwxyz0123456789")
	write("abcdefghijklmnopqrstuvwxyz0123456789")

	tag := now.Format("20060102")
	first := filepath.Join(dir, "xbase."+tag+".log")
	second := filepath.Join(dir, "xbase."+tag+".1.log")

	if _, err := os.Stat(first); err != nil {
		t.Fatalf("expected first file %s: %v", first, err)
	}

	if _, err := os.Stat(second); err != nil {
		t.Fatalf("expected second file %s: %v", second, err)
	}
}

func TestSyncerCleanupExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xbase.log")
	s := NewSyncer(
		WithPath(path),
		WithRotate(RotateDay),
		WithMaxAge(24*time.Hour),
	)
	defer func() { _ = s.Close() }()

	now := time.Now()
	currentTag := now.Format("20060102")
	oldTag := now.Add(-48 * time.Hour).Format("20060102")

	currentPath := filepath.Join(dir, "xbase."+currentTag+".log")
	oldPath := filepath.Join(dir, "xbase."+oldTag+".log")

	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatalf("write old file failed: %v", err)
	}

	oldTime := now.Add(-48 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes old file failed: %v", err)
	}

	s.fileTag = currentTag
	s.fileVersion = 0
	if err := s.cleanupExpiredFiles(now); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old file removed, got err=%v", err)
	}

	if _, err := os.Stat(currentPath); err != nil {
		t.Fatalf("expected current file kept: %v", err)
	}
}
