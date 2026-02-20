package password

// Hasher is an interface for hashing and comparing passwords.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}
