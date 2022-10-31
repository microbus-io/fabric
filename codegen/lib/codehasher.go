package lib

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

// SourceCodeSHA256 generates a hash of the source code files in the current directory and all sub-directories.
func SourceCodeSHA256() (string, error) {
	h := sha256.New()
	err := hashDir(h, ".")
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
			if fileName == "data" || fileName == "testdata" {
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
			fileName == "debug.test" ||
			fileName == "version-gen.go" {
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
