package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// SourceCodeSHA256 generates a SHA256 of the source code files in the indicated directory and its sub-directories.
// The directory is interpreted relative to the current working directory.
// Use "." to hash the current working directory.
func SourceCodeSHA256(directory string) (string, error) {
	h := sha256.New()
	err := hashDir(h, directory)
	if err != nil {
		return "", errors.Trace(err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashDir(h hash.Hash, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return errors.Trace(err)
	}
	for _, file := range files {
		fileName := filepath.Join(dir, file.Name())
		if file.IsDir() {
			if file.Name() == "data" || file.Name() == "testdata" {
				continue
			}
			err = hashDir(h, fileName)
			if err != nil {
				return errors.Trace(err)
			}
			continue
		}
		if strings.HasSuffix(file.Name(), "_test.go") ||
			strings.HasPrefix(file.Name(), ".") ||
			file.Name() == "debug.test" ||
			file.Name() == "version-gen.go" {
			continue
		}
		f, err := os.Open(fileName)
		if err != nil {
			return errors.Trace(err)
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
