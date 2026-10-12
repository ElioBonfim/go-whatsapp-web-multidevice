package usecase

import (
	"os"
	"path/filepath"
	"regexp"
)

var unsafeFileChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// downloadCacheBase names the file of one message's media: the same message always
// lands on the same name, so a repeated download request can reuse it instead of
// writing one more copy (each request used to leave a new <epoch>-<uuid> file).
func downloadCacheBase(messageID string) string {
	return unsafeFileChars.ReplaceAllString(messageID, "_")
}

// cachedDownload returns the file already downloaded for this message in dir, or ""
// when there is none. A file with the wrong size (a download cut short) does not count.
func cachedDownload(dir, messageID string, expectedSize uint64) string {
	matches, _ := filepath.Glob(filepath.Join(dir, downloadCacheBase(messageID)+".*"))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			continue
		}
		if expectedSize > 0 && uint64(info.Size()) != expectedSize {
			continue
		}
		return path
	}
	return ""
}

// keepAsCached renames a fresh download to the message's stable name, keeping the
// extension. On failure the fresh path is still valid and is returned as is.
func keepAsCached(path, messageID string) string {
	stable := filepath.Join(filepath.Dir(path), downloadCacheBase(messageID)+filepath.Ext(path))
	if stable == path {
		return path
	}
	if err := os.Rename(path, stable); err != nil {
		return path
	}
	return stable
}
