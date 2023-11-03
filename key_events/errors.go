package key_events

// Represents errors in this package.
type errCode int

const (
	// Failed to open the config file
	ErrOpenConfig errCode = iota
	// Failed to read the config file
	ErrReadFile
	// Config line is missing some arguments (or has too many arguments)
	ErrConfigArgsBad
	// Missing token "ch=" for channel
	ErrConfigChannelTokenMissing
	// Invalid channel, must be a value between 0 and 15
	ErrConfigChannelTokenInvalid
	// Missing token "ev=" for event
	ErrConfigEventTokenMissing
	// Invalid event, must be a value between 0 and 255
	ErrConfigEventInvalid
	// Missing token "key=" for key
	ErrConfigKeyTokenMissing
	// Invalid key
	ErrConfigKeyInvalid
	// Invalid action, must one of BASIC, VELOCITY, TOGGLE, REPEAT
	ErrConfigActionInvalid
	// Invalid action argument, expect a positive integer
	ErrConfigActionArgumentInvalid
	// Missing token "thres=" for threshold
	ErrConfigThresholdTokenMissing
	// Invalid event, must be a value between 0 and 255
	ErrConfigThresholdInvalid
	// The parsed value was ignored
	ErrConfigIgnored
)

// Implements the 'error' interface for 'errCode'.
func (e errCode) Error() string {
	switch e {
	case ErrOpenConfig:
		return "(key_events) failed to open the config file"
	case ErrReadFile:
		return "(key_events) failed to read the config file"
	case ErrConfigArgsBad:
		return "(key_events) config line is missing some arguments (or has too many arguments)"
	case ErrConfigChannelTokenMissing:
		return `(key_events) missing token "ch=" for channel`
	case ErrConfigChannelTokenInvalid:
		return `(key_events) invalid channel, must be a value between 0 and 15`
	case ErrConfigEventTokenMissing:
		return `(key_events) missing token "ev=" for event`
	case ErrConfigEventInvalid:
		return `(key_events) invalid event, must be a value between 0 and 255`
	case ErrConfigKeyTokenMissing:
		return `(key_events) missing token "key=" for key`
	case ErrConfigKeyInvalid:
		return `(key_events) invalid key`
	case ErrConfigActionInvalid:
		return `(key_events) invalid action, must one of BASIC, VELOCITY, TOGGLE, REPEAT`
	case ErrConfigActionArgumentInvalid:
		return "(key_events) invalid action argument, expect a positive integer"
	case ErrConfigThresholdTokenMissing:
		return `(key_events) missing token "thres=" for threshold`
	case ErrConfigThresholdInvalid:
		return "(key_events) invalid event, must be a value between 0 and 255"
	case ErrConfigIgnored:
		return "(key_events) the parsed value was ignored"
	default:
		return "(key_events) unknown error"
	}
}
