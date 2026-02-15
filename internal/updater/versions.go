package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/adrg/xdg"
)

// VersionEntry records a single archived binary.
type VersionEntry struct {
	Version    string    `json:"version"`
	ArchivedAt time.Time `json:"archived_at"`
	Filename   string    `json:"filename"`
}

// VersionArchive is the on-disk list of archived versions.
type VersionArchive struct {
	Versions []VersionEntry `json:"versions"`
}

// archiveDir returns $XDG_DATA_HOME/firecommit/versions/.
func archiveDir() string {
	return filepath.Join(xdg.DataHome, "firecommit", "versions")
}

// archivePath returns the path to versions.json.
func archivePath() string {
	return filepath.Join(archiveDir(), "versions.json")
}

// LoadArchive reads versions.json from disk. Returns an empty archive if the
// file does not exist.
func LoadArchive() (*VersionArchive, error) {
	data, err := os.ReadFile(archivePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &VersionArchive{}, nil
		}
		return nil, err
	}
	var a VersionArchive
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// SaveArchive writes the archive index to disk.
func (a *VersionArchive) SaveArchive() error {
	if err := os.MkdirAll(archiveDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(archivePath(), data, 0o644)
}

// filenameForVersion returns the binary filename used in the archive directory.
func filenameForVersion(version string) string {
	name := "firecommit-" + version
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// ArchiveCurrentBinary copies the running executable into the archive directory
// and records it in versions.json. If the version is already archived, it is a
// no-op.
func ArchiveCurrentBinary(version string) error {
	archive, err := LoadArchive()
	if err != nil {
		return fmt.Errorf("load archive: %w", err)
	}

	// Skip if already archived.
	for _, e := range archive.Versions {
		if e.Version == version {
			return nil
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determine executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	filename := filenameForVersion(version)
	dst := filepath.Join(archiveDir(), filename)

	if err := os.MkdirAll(archiveDir(), 0o755); err != nil {
		return err
	}
	if err := copyFile(execPath, dst); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dst, 0o755)
	}

	archive.Versions = append(archive.Versions, VersionEntry{
		Version:    version,
		ArchivedAt: time.Now(),
		Filename:   filename,
	})

	return archive.SaveArchive()
}

// PruneArchive removes the oldest entries so that at most keep versions remain.
func PruneArchive(keep int) error {
	archive, err := LoadArchive()
	if err != nil {
		return err
	}
	if len(archive.Versions) <= keep {
		return nil
	}

	// Oldest entries are at the front of the slice.
	toRemove := archive.Versions[:len(archive.Versions)-keep]
	archive.Versions = archive.Versions[len(archive.Versions)-keep:]

	for _, e := range toRemove {
		p := filepath.Join(archiveDir(), e.Filename)
		_ = os.Remove(p) // best-effort
	}

	return archive.SaveArchive()
}

// ListArchive returns the archived versions (newest last).
func ListArchive() ([]VersionEntry, error) {
	archive, err := LoadArchive()
	if err != nil {
		return nil, err
	}
	return archive.Versions, nil
}

// RestoreBinary replaces the current executable with the archived binary for
// the given version, using the same atomic-rename pattern as SelfUpdate.
func RestoreBinary(version string) error {
	archive, err := LoadArchive()
	if err != nil {
		return err
	}

	var entry *VersionEntry
	for i := range archive.Versions {
		if archive.Versions[i].Version == version {
			entry = &archive.Versions[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("version %s not found in archive", version)
	}

	src := filepath.Join(archiveDir(), entry.Filename)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("archived binary missing: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determine executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	// Atomic replace: rename old, copy archived, remove old.
	oldPath := execPath + ".old"
	if err := os.Rename(execPath, oldPath); err != nil {
		return fmt.Errorf("move current binary: %w", err)
	}

	if err := copyFile(src, execPath); err != nil {
		_ = os.Rename(oldPath, execPath) // try to restore
		return fmt.Errorf("install archived binary: %w", err)
	}
	os.Remove(oldPath)

	if runtime.GOOS != "windows" {
		_ = os.Chmod(execPath, 0o755)
	}

	// Recreate symlinks.
	binDir := filepath.Dir(execPath)
	baseName := filepath.Base(execPath)
	recreateLinks(binDir, baseName)

	return nil
}
