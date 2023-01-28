package key_handler

// Represents errors in this package.
type errCode int

const (
	// Failed to start the key generator
	ErrGetKeyGenerator errCode = iota
)

// Implements the 'error' interface for 'errCode'.
func (e errCode) Error() string {
	switch e {
	case ErrGetKeyGenerator:
		return "(key_handler) failed to start the key generator"
	default:
		return "(key_handler) unknown error"
	}
}
