package utils

import (
	"path/filepath"
	"strings"
	"unicode"

	"sapphirebroking.com/sftp_service/internal/apperrors"
)

// SanitizeName validates and normalises a user-supplied file/folder name.
// It rejects path separators, traversal sequences and control characters so
// names can never influence physical storage paths.
func SanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", apperrors.ErrInvalidName
	}
	if len(name) > 255 {
		return "", apperrors.ErrInvalidName
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return "", apperrors.ErrInvalidName
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", apperrors.ErrInvalidName
		}
	}
	return name, nil
}

// FileExtension returns the lower-cased extension without the dot ("" if none).
func FileExtension(name string) string {
	ext := filepath.Ext(name)
	return strings.ToLower(strings.TrimPrefix(ext, "."))
}
