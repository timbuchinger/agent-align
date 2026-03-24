package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"agent-align/internal/config"
)

// archiveSubdirectories creates a zip archive for each immediate subdirectory of
// target.Source, placing the resulting zip files in target.Destination. Each zip
// file is named after its source directory (e.g. "mydir" → "mydir.zip") and
// contains the full recursive contents of that directory.
func archiveSubdirectories(target config.ArchiveTarget) (int, error) {
	sourceInfo, err := os.Stat(target.Source)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect %s: %w", target.Source, err)
	}
	if !sourceInfo.IsDir() {
		return 0, fmt.Errorf("archive source %s is not a directory", target.Source)
	}

	entries, err := os.ReadDir(target.Source)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory %s: %w", target.Source, err)
	}

	if err := os.MkdirAll(target.Destination, 0o755); err != nil {
		return 0, fmt.Errorf("failed to create destination directory %s: %w", target.Destination, err)
	}

	var count int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(target.Source, entry.Name())
		zipPath := filepath.Join(target.Destination, entry.Name()+".zip")
		if err := createZipArchive(dirPath, zipPath); err != nil {
			return count, fmt.Errorf("failed to archive %s to %s: %w", dirPath, zipPath, err)
		}
		count++
	}
	return count, nil
}

// createZipArchive writes the recursive contents of the directory at source into
// a new zip file at zipPath. File paths inside the zip are relative to source.
func createZipArchive(source, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file %s: %w", zipPath, err)
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		f, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %w", rel, err)
		}

		in, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", path, err)
		}
		defer in.Close()

		if _, err := io.Copy(f, in); err != nil {
			return fmt.Errorf("failed to write zip entry %s: %w", rel, err)
		}
		return nil
	})
}
