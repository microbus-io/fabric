/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package yellowpagesapi

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql/driver"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	cipherEnabled = true
	cipherKey     = []byte("dd3BsEJFmxl7KlYTeFG5lGdyrrSa3iDA")
	cipherNonce   = []byte("0Fc+suW1xNly77gyfIe/zemsuVUdDcrE")
	cipherPtr     atomic.Pointer[cipher.AEAD]
)

// PersonKey is the ID of the person.
// The delegate directive routes dv8 directives declared on a key-typed field to the ID.
type PersonKey struct {
	ID int `json:"id,omitzero" dv8:"delegate,val>=0"`
}

// ParseKey returns a key from its scrambled (string) or unscrambled (int) ID.
// A zero key is returned if the ID can't be parsed.
func ParseKey(id any) PersonKey {
	switch v := id.(type) {
	case int:
		return PersonKey{ID: v}
	case int64:
		return PersonKey{ID: int(v)}
	case string:
		return PersonKey{ID: keyDecrypt(v)}
	default:
		return PersonKey{ID: 0}
	}
}

// Equal checks two keys for equality.
func (k PersonKey) Equal(other PersonKey) bool {
	return k.ID == other.ID
}

// String returns the ID of the key scrambled.
func (k PersonKey) String() string {
	return keyEncrypt(k.ID)
}

// IsZero tests if the ID is zero. Valid IDs starts from 1.
func (k PersonKey) IsZero() bool {
	return k.ID <= 0
}

// MarshalJSON overrides JSON serialization to encrypt the ID.
func (k PersonKey) MarshalJSON() (b []byte, err error) {
	return fmt.Appendf(b, `"%s"`, k.String()), nil
}

// UnmarshalJSON overrides JSON deserialization to decrypt the ID.
func (k *PersonKey) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	k.ID = keyDecrypt(s)
	return nil
}

// Scan implements the Scanner interface.
func (k *PersonKey) Scan(value interface{}) error {
	switch m := value.(type) {
	case []uint8:
		// Happens when query has no ? arguments
		s := string(m)
		k.ID, _ = strconv.Atoi(s)
	case int64:
		k.ID = int(m)
	default:
		k.ID = 0
	}
	return nil
}

// Value implements the driver Valuer interface.
func (k PersonKey) Value() (driver.Value, error) {
	if k.IsZero() {
		return nil, nil
	}
	return int64(k.ID), nil
}

// keyCipher creates the symmetric cipher singleton.
func keyCipher() cipher.AEAD {
	pgcm := cipherPtr.Load()
	if pgcm == nil {
		block, err := aes.NewCipher(cipherKey)
		if err != nil {
			panic(err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			panic(err)
		}
		cipherPtr.Store(&gcm)
		return gcm
	}
	return *pgcm
}

// keyEncrypt transforms the ID of the key into an encrypted 32 byte string.
func keyEncrypt(value int) (encrypted string) {
	if value <= 0 {
		return ""
	}
	if !cipherEnabled {
		return strconv.Itoa(value)
	}
	plainText := make([]byte, 8)
	binary.LittleEndian.PutUint64(plainText[:], uint64(value))
	gcm := keyCipher()
	nonceSize := gcm.NonceSize()
	dst := make([]byte, 0, 64)
	cipherText := gcm.Seal(dst, cipherNonce[:nonceSize], plainText[:], nil)
	return base64.RawURLEncoding.EncodeToString(cipherText)
}

// keyDecrypt transforms the encrypted string representation of the key back to an integer.
func keyDecrypt(encrypted string) (value int) {
	if !cipherEnabled {
		value, _ = strconv.Atoi(encrypted)
		return value
	}
	if len(encrypted) != 32 {
		return 0
	}
	cipherText, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return 0
	}
	gcm := keyCipher()
	nonceSize := gcm.NonceSize()
	plainText, err := gcm.Open(nil, cipherNonce[:nonceSize], cipherText, nil)
	if err != nil {
		return 0
	}
	return int(binary.LittleEndian.Uint64(plainText))
}
