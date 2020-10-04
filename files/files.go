package files

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"go-torrent-toy-system/util"
)

// FileInfo returns file size in bytes & sha256 signature
func FileInfo(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer util.Close(file)

	hash := sha256.New()
	n, err := io.Copy(hash, file)

	if err != nil {
		return 0, "", err
	}

	sig := fmt.Sprintf("%x", hash.Sum(nil))

	return n, sig, nil
}

// CreateEmptyFile creates an empty file in given size
func CreateEmptyFile(path string, size int64) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer util.Close(file)

	if _, err := file.Seek(size-1, io.SeekStart); err != nil {
		return err
	}

	if _, err := file.Write([]byte{0}); err != nil {
		return err
	}

	return nil
}

// WriteAt writes provided data at a given location in  a file
func WriteAt(path string, offset int64, data []byte) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer util.Close(file)

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	_, err = file.Write(data)

	return err
}

// ReadAt reads a chunk of data from file info buf
func ReadAt(path string, offset int64, buf []byte) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer util.Close(file)

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	_, err = io.ReadFull(file, buf)

	return err
}

// IsFile return true if path is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Mode().IsRegular()
}
