package scanner

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// normalizePath makes paths consistent for JSON output.
func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// SanitizeFilename replaces characters invalid in filenames.
func SanitizeFilename(s string) string { return sanitizeFilename(s) }

func sanitizeFilename(s string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range invalid {
		s = strings.ReplaceAll(s, c, "_")
	}
	return s
}

// MoveFile is the exported entry point used by the poller.
func MoveFile(src, dst, libraryRoot string) error { return moveFile(src, dst, libraryRoot, nil) }

// MoveFileWithProgress is like MoveFile, but calls onProgress with the
// cumulative bytes copied so far whenever the move falls back to a
// cross-device copy (onProgress is never called for a same-filesystem
// os.Rename, since that's effectively instant). onProgress may be nil.
func MoveFileWithProgress(src, dst, libraryRoot string, onProgress func(bytesDone int64)) error {
	return moveFile(src, dst, libraryRoot, onProgress)
}

// ComputeCRC32 hashes a file's content and returns its checksum as an
// uppercase 8-character hex string, matching the format used in filenames
// and metadata (e.g. "CA3F14A8"). Used by manual import, where the filename
// can't be trusted to carry the correct CRC.
func ComputeCRC32(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%08X", h.Sum32()), nil
}

// moveFile moves src to dst using a .tmp staging directory inside libraryRoot.
// It handles cross-device moves (e.g. downloads on local disk, library on NAS)
// by falling back to copy+delete when os.Rename returns EXDEV. onProgress, if
// non-nil, is called with cumulative bytes copied during that fallback.
func moveFile(src, dst, libraryRoot string, onProgress func(bytesDone int64)) error {
	tmpDir := filepath.Join(libraryRoot, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	tmpPath := filepath.Join(tmpDir, filepath.Base(src))

	// Stage: move src → .tmp (may cross device boundary).
	if err := os.Rename(src, tmpPath); err != nil {
		if isEXDEV(err) {
			if err := copyFile(src, tmpPath, onProgress); err != nil {
				return err
			}
			os.Remove(src)
		} else {
			return err
		}
	}

	// Final rename within the same volume (always same-device).
	return os.Rename(tmpPath, dst)
}

// copyBufSize is chosen to give reasonably granular progress updates without
// making excessive small syscalls on multi-GB video files.
const copyBufSize = 4 * 1024 * 1024

func copyFile(src, dst string, onProgress func(bytesDone int64)) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, copyBufSize)
	var done int64
	for {
		n, readErr := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			done += int64(n)
			if onProgress != nil {
				onProgress(done)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return out.Sync()
}

func isEXDEV(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return errors.Is(linkErr.Err, syscall.EXDEV)
	}
	return false
}
