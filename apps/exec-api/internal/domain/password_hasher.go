package domain

// PasswordHasher hashes and verifies passwords. Defined here (not imported
// from the password package) so the domain doesn't depend on a crypto
// library -- same pattern as TradeProducer.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	// Verify returns nil if plain matches hash, an error otherwise.
	Verify(hash, plain string) error
}
