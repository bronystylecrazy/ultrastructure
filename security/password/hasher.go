package password

import "golang.org/x/crypto/bcrypt"

// BcryptHasher is a bcrypt implementation of the Hasher interface.
type BcryptHasher struct{}

// NewBcryptHasher creates a new BcryptHasher.
func NewBcryptHasher() Hasher {
	return &BcryptHasher{}
}

// Hash hashes a password using bcrypt.
func (h *BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Compare compares a hashed password with a password.
func (h *BcryptHasher) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
