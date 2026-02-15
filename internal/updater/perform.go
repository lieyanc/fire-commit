package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SelfUpdate downloads and installs the latest release when it is newer than
// the current binary.
func SelfUpdate(ctx context.Context, currentVersion, channel string) error {
	return selfUpdate(ctx, currentVersion, channel, false)
}

// SelfUpdateForce always downloads and reinstalls the latest release on the
// selected channel, even if the current version appears up to date.
func SelfUpdateForce(ctx context.Context, currentVersion, channel string) error {
	return selfUpdate(ctx, currentVersion, channel, true)
}

// selfUpdate downloads and installs a release, replacing the current binary.
// The channel parameter determines which releases to consider ("latest" or "stable").
func selfUpdate(ctx context.Context, currentVersion, channel string, force bool) error {
	fmt.Println("Checking for updates...")

	release, err := FetchLatestRelease(ctx, channel)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := release.Version()

	if !force && !HasNewerVersion(currentVersion, latestVersion, channel) {
		fmt.Printf("Already up to date (%s).\n", currentVersion)
		return nil
	}

	if force {
		fmt.Printf("Forcing reinstall %s -> %s\n", currentVersion, latestVersion)
	} else {
		fmt.Printf("Updating %s -> %s\n", currentVersion, latestVersion)
	}

	// Find matching asset
	asset := FindAssetForPlatform(release.Assets)
	if asset == nil {
		return fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download archive
	fmt.Printf("Downloading %s...\n", asset.Name)
	archivePath, err := downloadToTemp(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(archivePath)

	fmt.Println("Verifying checksum...")
	if err := verifyDownloadedAsset(ctx, release, asset, archivePath); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// Extract to temp dir
	extractDir, err := os.MkdirTemp("", "fire-commit-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	if strings.HasSuffix(asset.Name, ".zip") {
		err = extractZip(archivePath, extractDir)
	} else {
		err = extractTarGz(archivePath, extractDir)
	}
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Find the extracted binary
	binaryName := "firecommit"
	if runtime.GOOS == "windows" {
		binaryName = "firecommit.exe"
	}
	newBinary, err := findFile(extractDir, binaryName)
	if err != nil {
		return fmt.Errorf("could not find %s in archive: %w", binaryName, err)
	}

	// Archive current binary before replacing (failure is non-fatal).
	if err := ArchiveCurrentBinary(currentVersion); err != nil {
		fmt.Printf("Warning: could not archive current binary: %v\n", err)
	}

	appliedNow, err := replaceCurrentBinary(newBinary)
	if err != nil {
		return fmt.Errorf("failed to install updated binary: %w", err)
	}
	if appliedNow {
		fmt.Printf("Successfully updated to %s\n", latestVersion)
	} else {
		fmt.Printf("Update to %s is staged and will complete after process exit.\n", latestVersion)
	}

	// Prune old archives to keep at most 5 versions (failure is non-fatal).
	if err := PruneArchive(5); err != nil {
		fmt.Printf("Warning: could not prune version archive: %v\n", err)
	}

	return nil
}

func downloadToTemp(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "fire-commit-*.download")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, filepath.Base(hdr.Name))

		switch hdr.Typeflag {
		case tar.TypeReg:
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		target := filepath.Join(destDir, filepath.Base(f.Name))
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func findFile(dir, name string) (string, error) {
	var found string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("file %s not found", name)
	}
	return found, nil
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
	return out.Close()
}

func recreateLinks(binDir, baseName string) {
	links := []string{"fcmt", "git-fire-commit"}
	for _, link := range links {
		linkPath := filepath.Join(binDir, link)
		if runtime.GOOS == "windows" {
			linkPath += ".exe"
			// On Windows, copy instead of symlink
			_ = copyFile(filepath.Join(binDir, baseName), linkPath)
		} else {
			os.Remove(linkPath)
			_ = os.Symlink(baseName, linkPath)
		}
	}
}
