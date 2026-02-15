package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// verifyDownloadedAsset validates the archive against checksums.txt from the
// same release.
func verifyDownloadedAsset(ctx context.Context, release *Release, asset *Asset, archivePath string) error {
	if release == nil || asset == nil {
		return fmt.Errorf("invalid release metadata")
	}

	checksumAsset := findChecksumsAsset(release.Assets)
	if checksumAsset == nil {
		return fmt.Errorf("checksums.txt not found in release assets")
	}

	checksumPath, err := downloadToTemp(ctx, checksumAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	defer os.Remove(checksumPath)

	expected, err := checksumForAsset(checksumPath, asset.Name)
	if err != nil {
		return err
	}

	actual, err := sha256File(archivePath)
	if err != nil {
		return fmt.Errorf("compute sha256: %w", err)
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("sha256 mismatch for %s", asset.Name)
	}
	return nil
}

func findChecksumsAsset(assets []Asset) *Asset {
	for i := range assets {
		if assets[i].Name == "checksums.txt" {
			return &assets[i]
		}
	}
	return nil
}

func checksumForAsset(path, assetName string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		sum := strings.ToLower(fields[0])
		filename := strings.TrimPrefix(fields[1], "*")
		if filename != assetName {
			continue
		}

		if len(sum) != 64 || !isHex(sum) {
			return "", fmt.Errorf("invalid checksum format for %s", assetName)
		}
		return sum, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum entry not found for %s", assetName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return s != ""
}
