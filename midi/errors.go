package midi

// Represents errors in this package.
type errCode int

const (
	// Failed to list the available MIDI devices
	ErrListDevices errCode = iota
	// Failed to open the requested device
	ErrOpenDevice
	// Failed to listen to the requested device
	ErrListenDevice
)

// Implements the 'error' interface for 'errCode'.
func (e errCode) Error() string {
	switch e {
	case ErrListDevices:
		return "(midi) failed to list the available MIDI devices"
	case ErrOpenDevice:
		return "(midi) failed to open the requested device"
	case ErrListenDevice:
		return "(midi) failed to listen to the requested device"
	default:
		return "(midi) unknown error"
	}
}
