package directoryapi

import (
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
)

// Validate validates the field of the person.
// First and last name and email are required. Optional birthday must be in the past.
func (person *Person) Validate() error {
	person.FirstName = strings.TrimSpace(person.FirstName)
	person.LastName = strings.TrimSpace(person.LastName)
	person.Email = strings.TrimSpace(person.Email)
	if person.FirstName == "" || person.LastName == "" {
		return errors.New("names cannot be empty")
	}
	if person.Email == "" {
		return errors.New("email cannot be empty")
	}
	if !person.Birthday.IsZero() && person.Birthday.After(time.Now()) {
		return errors.New("birthday must be a past date")
	}
	return nil
}
