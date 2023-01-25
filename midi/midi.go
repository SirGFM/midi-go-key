package midi

import (
	"fmt"
	"io"
	"sync/atomic"

	"github.com/SirGFM/midi-go-key/err_wrap"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/midicatdrv"
)

// The maximum velocity reported by the driver for any given MIDI event.
const MaxVelocity = 128

type Midi interface {
	// Close releases resources associated with this device.
	Close() error
}

// Cleanup releases global resources instantiated by the midi package.
// It should be called only once as the application is exiting.
func Cleanup() {
	midi.CloseDriver()
}

// Types of recognized MIDI events.
type MidiEventType int

const (
	// Event that isn't handled by the package.
	EventUnknown MidiEventType = iota
	// Note On event (0x9x xx xx ...)
	EventNoteOn
	// Note Off event (0x8x xx xx ...)
	EventNoteOff
)

func (evType MidiEventType) String() string {
	switch evType {
	case EventUnknown:
		return "EventUnknown"
	case EventNoteOn:
		return "EventNoteOn"
	case EventNoteOff:
		return "EventNoteOff"
	default:
		return "Invalid MidiEventType"
	}
}

// Convert the MIDI event to its byte representation.
func (evType MidiEventType) ToUint8() uint8 {
	switch evType {
	case EventUnknown:
		return 0x00
	case EventNoteOn:
		return 0x90
	case EventNoteOff:
		return 0x80
	default:
		return 0x00
	}
}

// A MIDI event.
type MidiEvent struct {
	// The original MIDI message.
	Source []byte
	// The timestamp when the MIDI event was generated, in milliseconds.
	Timestamp int32
	// The message's type
	Type MidiEventType
	// The message's channel.
	Channel uint8
	// The message's key.
	Key uint8
	// The message's velocity.
	Velocity uint8
}

// Convert the MIDI event to a string.
func (ev MidiEvent) String() string {
	return fmt.Sprintf(
		"% 16d: %x - chan: %x - key: %x - vel: %d - type: %s",
		ev.Timestamp,
		ev.Source,
		ev.Channel,
		ev.Key,
		ev.Velocity,
		ev.Type,
	)
}

// An implementation of a MIDI device.
type midiDev struct {
	// The internal device, acquired from gomidi/midi.
	dev io.Closer
	// Channel used to send a received MIDI event.
	sender chan MidiEvent
	// A function used to stop this midi device.
	stop func()
	// Whether the device has already been stopped.
	stopped int32
}

// isClosed returns or whether or not this device is closed.
func (m *midiDev) isClosed() bool {
	return atomic.LoadInt32(&m.stopped) != 0
}

func (m *midiDev) Close() error {
	if m.isClosed() {
		return nil
	}

	m.stop()
	err := m.dev.Close()
	atomic.CompareAndSwapInt32(&m.stopped, 0, 1)
	close(m.sender)
	return err
}

// recv handles received messages, forwarding them to the configured channel.
func (m *midiDev) recv(msg midi.Message, timestampMs int32) {
	// Copy the received message to avoid issues caused by
	// referencing a buffer maintained by the internal package.
	ev := MidiEvent{
		Source:    append([]byte{}, []byte(msg)...),
		Timestamp: timestampMs,
	}

	switch {
	case msg.GetNoteOn(&ev.Channel, &ev.Key, &ev.Velocity):
		ev.Type = EventNoteOn
	case msg.GetNoteOff(&ev.Channel, &ev.Key, &ev.Velocity):
		ev.Type = EventNoteOff
	default:
		ev.Type = EventUnknown
	}

	// Send the event to the handler.
	if !m.isClosed() {
		m.sender <- ev
	}
}

// NewMidi opens a new MIDI device.
func NewMidi(port int, conn chan MidiEvent) (Midi, error) {
	in, err := midi.InPort(port)
	if err != nil {
		return nil, err_wrap.Wrap(err, ErrOpenDevice)
	}

	dev := &midiDev{
		dev:    in,
		sender: conn,
	}

	stop, err := midi.ListenTo(in, dev.recv)
	if err != nil {
		return nil, err_wrap.Wrap(err, ErrListDevices)
	}

	dev.stop = stop
	return dev, nil
}

// A MIDI device.
type Device struct {
	// The device's port number
	Port int
	// The device's name
	Name string
}

// ListDevices lists every connected device.
func ListDevices() ([]Device, error) {
	ins, err := drivers.Ins()
	if err != nil {
		return nil, err_wrap.Wrap(err, ErrListDevices)
	}

	var devs []Device
	for _, drv := range ins {
		dev := Device{
			Port: drv.Number(),
			Name: drv.String(),
		}

		devs = append(devs, dev)
	}

	return devs, nil
}
