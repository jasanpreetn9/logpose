package scanner

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"Whisky Peak":           "Whisky Peak",
		"Question: Answer":      "Question_ Answer",
		"a/b\\c:d*e?f\"g<h>i|j": "a_b_c_d_e_f_g_h_i_j",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComputeCRC32(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.bin")
	content := []byte("some file content to hash")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := ComputeCRC32(path)
	if err != nil {
		t.Fatalf("ComputeCRC32: %v", err)
	}

	wantHex := fmt.Sprintf("%08X", crc32.ChecksumIEEE(content))
	if got != wantHex {
		t.Errorf("ComputeCRC32 = %q, want %q", got, wantHex)
	}
	if len(got) != 8 {
		t.Errorf("expected 8-char hex string, got %q (len %d)", got, len(got))
	}
}

func TestComputeCRC32_MissingFile(t *testing.T) {
	if _, err := ComputeCRC32("/nonexistent/path/file.mkv"); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestMoveFile_SameFilesystem(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mkv")
	if err := os.WriteFile(src, []byte("video"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	dstDir := filepath.Join(dir, "arc")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dst := filepath.Join(dstDir, "dst.mkv")

	if err := MoveFile(src, dst, dir); err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected src to be gone, stat err = %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil || string(b) != "video" {
		t.Errorf("dst content = %q, err = %v", b, err)
	}
}

func TestMoveFileWithProgress_ReportsCumulativeBytesOnFallbackCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	content := make([]byte, copyBufSize*2+100) // spans multiple copy chunks
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	dst := filepath.Join(dir, "dst.bin")

	var lastReported int64
	if err := copyFile(src, dst, func(done int64) { lastReported = done }); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if lastReported != int64(len(content)) {
		t.Errorf("final progress = %d, want %d", lastReported, len(content))
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if len(got) != len(content) {
		t.Fatalf("copied %d bytes, want %d", len(got), len(content))
	}
	for i := range content {
		if got[i] != content[i] {
			t.Fatalf("content mismatch at byte %d", i)
		}
	}
}

func TestIsEXDEV(t *testing.T) {
	wrapped := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.EXDEV}
	if !isEXDEV(wrapped) {
		t.Error("expected isEXDEV to detect a wrapped syscall.EXDEV")
	}
	if isEXDEV(errors.New("some other error")) {
		t.Error("isEXDEV should be false for unrelated errors")
	}
	if isEXDEV(&os.LinkError{Err: os.ErrPermission}) {
		t.Error("isEXDEV should be false for a LinkError wrapping a different errno")
	}
}

// Regression test for #10/#11: a file whose size is still changing (still
// being written by the torrent client) must be reported unstable so callers
// leave it alone instead of importing (and deleting the source of) a
// partial download.
func TestIsFileStable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "growing.mkv")
	if err := os.WriteFile(path, []byte("initial"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	t.Run("stable file", func(t *testing.T) {
		if !isFileStable(path) {
			t.Error("expected a file with unchanged size/mtime to be stable")
		}
	})

	t.Run("growing file", func(t *testing.T) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			time.Sleep(fileStabilityWait / 4)
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			f.WriteString(" more data")
			f.Close()
		}()
		stable := isFileStable(path)
		<-done
		if stable {
			t.Error("expected a file that changed size mid-check to be unstable")
		}
	})
}

func TestIsFileStable_MissingFile(t *testing.T) {
	if isFileStable(filepath.Join(t.TempDir(), "missing.mkv")) {
		t.Error("expected a missing file to be reported unstable")
	}
}
