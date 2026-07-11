package busstopapi

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
	cipherKey     = []byte("_CIPHER_KEY_____________________")
	cipherNonce   = []byte("_CIPHER_NONCE___________________")
	cipherPtr     atomic.Pointer[cipher.AEAD]
)

// BusStopKey is the ID of the bus stop.
// The delegate directive routes dv8 directives declared on a key-typed field to the ID.
type BusStopKey struct {
	ID int `json:"id,omitzero" dv8:"delegate,val>=0"`
}

// ParseKey returns a key from its scrambled (string) or unscrambled (int) ID.
// A zero key is returned if the ID can't be parsed.
func ParseKey(id any) BusStopKey {
	switch v := id.(type) {
	case int:
		return BusStopKey{ID: v}
	case int64:
		return BusStopKey{ID: int(v)}
	case string:
		return BusStopKey{ID: keyDecrypt(v)}
	default:
		return BusStopKey{ID: 0}
	}
}

// Equal checks two keys for equality.
func (k BusStopKey) Equal(other BusStopKey) bool {
	return k.ID == other.ID
}

// String returns the ID of the key scrambled.
func (k BusStopKey) String() string {
	return keyEncrypt(k.ID)
}

// IsZero tests if the ID is zero. Valid IDs starts from 1.
func (k BusStopKey) IsZero() bool {
	return k.ID <= 0
}

// MarshalJSON overrides JSON serialization to encrypt the ID.
func (k BusStopKey) MarshalJSON() (b []byte, err error) {
	return fmt.Appendf(b, `"%s"`, k.String()), nil
}

// UnmarshalJSON overrides JSON deserialization to decrypt the ID.
func (k *BusStopKey) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	k.ID = keyDecrypt(s)
	return nil
}

// Scan implements the Scanner interface.
func (k *BusStopKey) Scan(value interface{}) error {
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
func (k BusStopKey) Value() (driver.Value, error) {
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
