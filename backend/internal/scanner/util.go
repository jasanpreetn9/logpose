package scanner

import (
	"errors"
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
func MoveFile(src, dst, libraryRoot string) error { return moveFile(src, dst, libraryRoot) }

// moveFile moves src to dst using a .tmp staging directory inside libraryRoot.
// It handles cross-device moves (e.g. downloads on local disk, library on NAS)
// by falling back to copy+delete when os.Rename returns EXDEV.
func moveFile(src, dst, libraryRoot string) error {
	tmpDir := filepath.Join(libraryRoot, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	tmpPath := filepath.Join(tmpDir, filepath.Base(src))

	// Stage: move src → .tmp (may cross device boundary).
	if err := os.Rename(src, tmpPath); err != nil {
		if isEXDEV(err) {
			if err := copyFile(src, tmpPath); err != nil {
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

func copyFile(src, dst string) error {
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

	if _, err := io.Copy(out, in); err != nil {
		return err
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
