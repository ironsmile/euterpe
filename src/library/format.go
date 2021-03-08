package library

import (
	"path/filepath"
	"strings"
)

func mediaFormatFromFileName(path string) string {
	format := strings.TrimLeft(filepath.Ext(path), ".")
	if format == "" {
		format = "mp3"
	}
	return strings.ToLower(format)
}
