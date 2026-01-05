package vault

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
)

type archiveEntry struct {
	Name  string
	Size  uint64
	IsDir bool
	Open  func() (io.ReadCloser, error)
}

func ReadLairHashesFromArchive(path string) (Hashes, error) {
	entries, closeFn, err := readArchiveEntries(path)
	if err != nil {
		return Hashes{}, err
	}
	defer closeFn()
	hashes, _, err := readLairHashes(entries)
	return hashes, err
}

func ComputeROMHashesFromArchive(path string) (Hashes, string, error) {
	entries, closeFn, err := readArchiveEntries(path)
	if err != nil {
		return Hashes{}, "", err
	}
	defer closeFn()
	return computeROMHashes(entries)
}

func ReadLairHashesFromZip(path string) (Hashes, error) {
	return ReadLairHashesFromArchive(path)
}

func ComputeROMHashesFromZip(path string) (Hashes, string, error) {
	return ComputeROMHashesFromArchive(path)
}

func readArchiveEntries(path string) ([]archiveEntry, func() error, error) {
	kind, err := detectArchiveKind(path)
	if err != nil {
		return nil, func() error { return nil }, err
	}
	switch kind {
	case "zip":
		return readZipEntries(path)
	case "7z":
		return readSevenZipEntries(path)
	default:
		return nil, func() error { return nil }, fmt.Errorf("unsupported archive format")
	}
}

func detectArchiveKind(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 6)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", err
	}
	buf = buf[:n]
	if len(buf) >= 6 && bytes.Equal(buf[:6], []byte{0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c}) {
		return "7z", nil
	}
	if len(buf) >= 4 {
		prefix := buf[:4]
		switch {
		case bytes.Equal(prefix, []byte("PK\x03\x04")),
			bytes.Equal(prefix, []byte("PK\x05\x06")),
			bytes.Equal(prefix, []byte("PK\x07\x08")):
			return "zip", nil
		}
	}
	return "", fmt.Errorf("unsupported archive format")
}

func readZipEntries(path string) ([]archiveEntry, func() error, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, func() error { return nil }, err
	}
	entries := make([]archiveEntry, 0, len(reader.File))
	for _, file := range reader.File {
		entry := archiveEntry{
			Name:  file.Name,
			Size:  file.UncompressedSize64,
			IsDir: file.FileInfo().IsDir(),
			Open:  file.Open,
		}
		entries = append(entries, entry)
	}
	return entries, reader.Close, nil
}

func readSevenZipEntries(path string) ([]archiveEntry, func() error, error) {
	reader, err := sevenzip.OpenReader(path)
	if err != nil {
		return nil, func() error { return nil }, err
	}
	entries := make([]archiveEntry, 0, len(reader.File))
	for _, file := range reader.File {
		entry := archiveEntry{
			Name:  file.Name,
			Size:  file.UncompressedSize,
			IsDir: file.FileInfo().IsDir(),
			Open:  file.Open,
		}
		entries = append(entries, entry)
	}
	return entries, reader.Close, nil
}

func readLairHashes(entries []archiveEntry) (Hashes, []byte, error) {
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		if strings.EqualFold(filepath.Base(entry.Name), "Vimm's Lair.txt") {
			rc, err := entry.Open()
			if err != nil {
				return Hashes{}, nil, err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return Hashes{}, nil, err
			}
			hashes, err := ParseLairTxt(data)
			return hashes, data, err
		}
	}
	return Hashes{}, nil, fmt.Errorf("Vimm's Lair.txt not found in archive")
}

func computeROMHashes(entries []archiveEntry) (Hashes, string, error) {
	candidates := make([]archiveEntry, 0, len(entries))
	candidateNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		if strings.EqualFold(filepath.Base(entry.Name), "Vimm's Lair.txt") {
			continue
		}
		candidates = append(candidates, entry)
		candidateNames = append(candidateNames, entry.Name)
	}
	if len(candidates) == 0 {
		return Hashes{}, "", fmt.Errorf("no ROM file found in archive")
	}

	var expectedName string
	if _, lairData, err := readLairHashes(entries); err == nil {
		expectedName = FindRomNameInLair(lairData, candidateNames)
	}

	var target *archiveEntry
	var maxSize uint64
	for i := range candidates {
		entry := candidates[i]
		name := filepath.Base(entry.Name)
		if expectedName != "" && strings.EqualFold(name, filepath.Base(expectedName)) {
			target = &entry
			break
		}
		if entry.Size >= maxSize {
			maxSize = entry.Size
			target = &entry
		}
	}
	if target == nil {
		return Hashes{}, "", fmt.Errorf("no ROM file found in archive")
	}

	rc, err := target.Open()
	if err != nil {
		return Hashes{}, "", err
	}
	defer rc.Close()

	md5Hash := md5.New()
	sha1Hash := sha1.New()
	crcHash := crc32.NewIEEE()
	writer := io.MultiWriter(md5Hash, sha1Hash, crcHash)
	if _, err := io.Copy(writer, rc); err != nil {
		return Hashes{}, "", err
	}

	return Hashes{
		CRC:  fmt.Sprintf("%08x", crcHash.Sum32()),
		MD5:  fmt.Sprintf("%x", md5Hash.Sum(nil)),
		SHA1: fmt.Sprintf("%x", sha1Hash.Sum(nil)),
	}, target.Name, nil
}
