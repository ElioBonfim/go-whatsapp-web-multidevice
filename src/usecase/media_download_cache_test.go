package usecase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCachedDownloadReusesTheSameMessage(t *testing.T) {
	dir := t.TempDir()
	if got := cachedDownload(dir, "3A637AD4/../x", 5); got != "" {
		t.Fatalf("empty dir: got %q", got)
	}

	fresh := filepath.Join(dir, "1791831252-uuid.mp4")
	if err := os.WriteFile(fresh, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	kept := keepAsCached(fresh, "3A637AD4/../x")
	if filepath.Base(kept) != "3A637AD4____x.mp4" || filepath.Dir(kept) != dir {
		t.Fatalf("stable name: got %q", kept)
	}
	if got := cachedDownload(dir, "3A637AD4/../x", 5); got != kept {
		t.Fatalf("reuse: got %q, want %q", got, kept)
	}
	if got := cachedDownload(dir, "3A637AD4/../x", 6); got != "" {
		t.Fatalf("a file of the wrong size is a cut download, not a cache hit: got %q", got)
	}
	if got := cachedDownload(dir, "3A637AD4/../x", 0); got != kept {
		t.Fatalf("unknown size: got %q", got)
	}
}
