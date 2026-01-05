package cli

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/emelander/vimm-cli/internal/vault"
)

func TestVerifyZipStrict(t *testing.T) {
	dir := t.TempDir()
	romName := "Test Game (USA).bin"
	romContent := []byte("verify-zip-test")
	hashes := computeHashes(romContent)

	zipPath := filepath.Join(dir, "verify.zip")
	if err := writeTestZip(zipPath, romName, romContent, hashes, true); err != nil {
		t.Fatalf("writeTestZip: %v", err)
	}

	if err := verifyZip(zipPath, hashes, true); err != nil {
		t.Fatalf("verifyZip error: %v", err)
	}
}

func TestVerifyZipMismatch(t *testing.T) {
	dir := t.TempDir()
	romName := "Test Game (USA).bin"
	romContent := []byte("verify-zip-mismatch")
	hashes := computeHashes(romContent)

	zipPath := filepath.Join(dir, "verify-mismatch.zip")
	if err := writeTestZip(zipPath, romName, romContent, hashes, true); err != nil {
		t.Fatalf("writeTestZip: %v", err)
	}

	bad := hashes
	bad.MD5 = "00000000000000000000000000000000"
	if err := verifyZip(zipPath, bad, true); err == nil {
		t.Fatalf("expected verifyZip to fail with bad hashes")
	}
}

func writeTestZip(path, romName string, romContent []byte, hashes vault.Hashes, includeLair bool) error {
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

func computeHashes(data []byte) vault.Hashes {
	crc := crc32.ChecksumIEEE(data)
	md5Sum := md5.Sum(data)
	sha1Sum := sha1.Sum(data)
	return vault.Hashes{
		CRC:  fmt.Sprintf("%08x", crc),
		MD5:  fmt.Sprintf("%x", md5Sum),
		SHA1: fmt.Sprintf("%x", sha1Sum),
	}
}
