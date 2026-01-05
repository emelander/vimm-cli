package vault

import (
	"path/filepath"
	"testing"
)

func TestReadLairHashesFromArchive7z(t *testing.T) {
	path := filepath.Join("testdata", "lair.7z")
	hashes, err := ReadLairHashesFromArchive(path)
	if err != nil {
		t.Fatalf("ReadLairHashesFromArchive error: %v", err)
	}
	if hashes.CRC != "e3112244" || hashes.MD5 != "41700f5a11bf8a50946a60ed1688e4c9" || hashes.SHA1 != "4ba4bfd48c861865c72c06ae4789deb4fd237e91" {
		t.Fatalf("unexpected hashes: %+v", hashes)
	}
}

func TestComputeROMHashesFromArchive7z(t *testing.T) {
	path := filepath.Join("testdata", "lair.7z")
	hashes, name, err := ComputeROMHashesFromArchive(path)
	if err != nil {
		t.Fatalf("ComputeROMHashesFromArchive error: %v", err)
	}
	if name != "Test Game (USA).bin" {
		t.Fatalf("expected rom name, got %q", name)
	}
	if hashes.CRC != "e3112244" || hashes.MD5 != "41700f5a11bf8a50946a60ed1688e4c9" || hashes.SHA1 != "4ba4bfd48c861865c72c06ae4789deb4fd237e91" {
		t.Fatalf("unexpected hashes: %+v", hashes)
	}
}
