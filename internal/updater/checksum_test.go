package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumForAsset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(path, []byte(
		"4bf5122f344554c53bde2ebb8cd2b7e3d1600ad631c385a5d7cce23c7785459a  fire-commit_dev_darwin_arm64.tar.gz\n",
	), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	got, err := checksumForAsset(path, "fire-commit_dev_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("checksumForAsset() error: %v", err)
	}
	want := "4bf5122f344554c53bde2ebb8cd2b7e3d1600ad631c385a5d7cce23c7785459a"
	if got != want {
		t.Fatalf("checksum mismatch: got %q want %q", got, want)
	}
}

func TestChecksumForAssetNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(path, []byte(
		"4bf5122f344554c53bde2ebb8cd2b7e3d1600ad631c385a5d7cce23c7785459a  another-file.tar.gz\n",
	), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	if _, err := checksumForAsset(path, "fire-commit_dev_darwin_arm64.tar.gz"); err == nil {
		t.Fatalf("checksumForAsset() expected error for missing asset")
	}
}
