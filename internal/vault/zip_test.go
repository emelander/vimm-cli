package vault

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

func TestZipHashHelpers(t *testing.T) {
	dir := t.TempDir()
	romName := "Test Game (USA).bin"
	romContent := []byte("unit-test-rom-data")
	hashes := computeHashes(romContent)

	zipPath := filepath.Join(dir, "test.zip")
	if err := writeTestZip(zipPath, romName, romContent, hashes, true); err != nil {
		t.Fatalf("writeTestZip: %v", err)
	}

	gotLair, err := ReadLairHashesFromZip(zipPath)
	if err != nil {
		t.Fatalf("ReadLairHashesFromZip error: %v", err)
	}
	if gotLair != hashes {
		t.Fatalf("expected lair hashes %+v, got %+v", hashes, gotLair)
	}

	gotROM, name, err := ComputeROMHashesFromZip(zipPath)
	if err != nil {
		t.Fatalf("ComputeROMHashesFromZip error: %v", err)
	}
	if name != romName {
		t.Fatalf("expected rom name %q, got %q", romName, name)
	}
	if gotROM != hashes {
		t.Fatalf("expected rom hashes %+v, got %+v", hashes, gotROM)
	}
}

func TestReadLairHashesFromZipMissing(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "missing.zip")
	if err := writeTestZip(zipPath, "Test.bin", []byte("x"), Hashes{}, false); err != nil {
		t.Fatalf("writeTestZip: %v", err)
	}
	if _, err := ReadLairHashesFromZip(zipPath); err == nil {
		t.Fatalf("expected error when Vimm's Lair.txt is missing")
	}
}

func writeTestZip(path, romName string, romContent []byte, hashes Hashes, includeLair bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	if includeLair {
		lairText := fmt.Sprintf("CRC:   %s\nMD5:   %s\nSHA-1: %s\n", hashes.CRC, hashes.MD5, hashes.SHA1)
		w, err := zw.Create("Vimm's Lair.txt")
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(lairText)); err != nil {
			return err
		}
	}
	w, err := zw.Create(romName)
	if err != nil {
		return err
	}
	if _, err := w.Write(romContent); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return nil
}

func computeHashes(data []byte) Hashes {
	crc := crc32.ChecksumIEEE(data)
	md5Sum := md5.Sum(data)
	sha1Sum := sha1.Sum(data)
	return Hashes{
		CRC:  fmt.Sprintf("%08x", crc),
		MD5:  fmt.Sprintf("%x", md5Sum),
		SHA1: fmt.Sprintf("%x", sha1Sum),
	}
}
