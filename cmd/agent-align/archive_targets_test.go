package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-align/internal/config"
)

func TestArchiveSubdirectoriesBasic(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")

	// Create two subdirectories with files
	if err := os.MkdirAll(filepath.Join(source, "alpha"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(source, "beta"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "alpha", "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "beta", "data.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	target := config.ArchiveTarget{Source: source, Destination: dest}
	count, err := archiveSubdirectories(target)
	if err != nil {
		t.Fatalf("archiveSubdirectories returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 zip files, got %d", count)
	}

	for _, name := range []string{"alpha.zip", "beta.zip"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}
}

func TestArchiveSubdirectoriesSkipsFiles(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")

	if err := os.MkdirAll(filepath.Join(source, "subdir"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	// Write a file directly under source (should be skipped)
	if err := os.WriteFile(filepath.Join(source, "root_file.txt"), []byte("skip me"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "subdir", "child.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	target := config.ArchiveTarget{Source: source, Destination: dest}
	count, err := archiveSubdirectories(target)
	if err != nil {
		t.Fatalf("archiveSubdirectories returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 zip file, got %d", count)
	}

	// Only subdir.zip should exist, not root_file.txt.zip
	entries, _ := os.ReadDir(dest)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in dest, got %d", len(entries))
	}
	if entries[0].Name() != "subdir.zip" {
		t.Errorf("expected subdir.zip, got %s", entries[0].Name())
	}
}

func TestArchiveSubdirectoriesZipContents(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")

	if err := os.MkdirAll(filepath.Join(source, "mydir", "nested"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "mydir", "top.txt"), []byte("top"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "mydir", "nested", "deep.txt"), []byte("deep"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	target := config.ArchiveTarget{Source: source, Destination: dest}
	if _, err := archiveSubdirectories(target); err != nil {
		t.Fatalf("archiveSubdirectories returned error: %v", err)
	}

	// Read the zip and verify its contents
	zipPath := filepath.Join(dest, "mydir.zip")
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	found := make(map[string]bool)
	for _, f := range r.File {
		found[f.Name] = true
	}

	for _, want := range []string{"top.txt", "nested/deep.txt"} {
		if !found[want] {
			t.Errorf("zip missing expected entry %q; found: %v", want, found)
		}
	}
}

func TestArchiveSubdirectoriesSourceNotDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	target := config.ArchiveTarget{Source: file, Destination: filepath.Join(dir, "dest")}
	_, err := archiveSubdirectories(target)
	if err == nil {
		t.Fatal("expected error for non-directory source")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchiveSubdirectoriesCreatesDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "new", "nested", "dest")

	if err := os.MkdirAll(filepath.Join(source, "subdir"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	target := config.ArchiveTarget{Source: source, Destination: dest}
	_, err := archiveSubdirectories(target)
	if err != nil {
		t.Fatalf("archiveSubdirectories returned error: %v", err)
	}

	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected destination directory to be created: %v", err)
	}
}

func TestCreateZipArchiveEmpty(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "empty")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	zipPath := filepath.Join(dir, "empty.zip")
	if err := createZipArchive(source, zipPath); err != nil {
		t.Fatalf("createZipArchive returned error: %v", err)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Errorf("expected zip file to exist: %v", err)
	}
}
