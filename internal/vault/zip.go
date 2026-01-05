package vault

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"io"
	"path/filepath"
	"strings"
)

func ReadLairHashesFromZip(path string) (Hashes, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Hashes{}, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Base(file.Name), "Vimm's Lair.txt") {
			rc, err := file.Open()
			if err != nil {
				return Hashes{}, err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return Hashes{}, err
			}
			return ParseLairTxt(data)
		}
	}
	return Hashes{}, fmt.Errorf("Vimm's Lair.txt not found in archive")
}

func ComputeROMHashesFromZip(path string) (Hashes, string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Hashes{}, "", err
	}
	defer reader.Close()

	var expectedName string
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Base(file.Name), "Vimm's Lair.txt") {
			rc, err := file.Open()
			if err != nil {
				return Hashes{}, "", err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return Hashes{}, "", err
			}
			expectedName = ExtractRomNameFromLair(data)
			break
		}
	}

	var target *zip.File
	var maxSize uint64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(file.Name)
		if strings.EqualFold(name, "Vimm's Lair.txt") {
			continue
		}
		if expectedName != "" && strings.EqualFold(name, filepath.Base(expectedName)) {
			target = file
			break
		}
		if file.UncompressedSize64 >= maxSize {
			maxSize = file.UncompressedSize64
			target = file
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
