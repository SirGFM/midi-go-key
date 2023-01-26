package key_events

import (
	"fmt"
	"runtime/debug"
	"testing"
	"time"

	"github.com/SirGFM/midi-go-key/midi"
)

// Mocks KeyController.
type mockKeyController map[int]bool

func (kc mockKeyController) Close() error {
	return nil
}

func (kc mockKeyController) PressKeys(keys ...int) {
	for _, key := range keys {
		kc[key] = true
	}
}

func (kc mockKeyController) ReleaseKeys(keys ...int) {
	for _, key := range keys {
		kc[key] = false
	}
}

// assert tests whether the given condition is true,
// printing the message and marking the test as having failed otherwise.
func assert(t *testing.T, condition bool, fmt string, args ...interface{}) {
	if !condition {
		debug.PrintStack()
		t.Fatalf(fmt, args...)
	}
}

// The last time an event was sent, for the event timestamp.
var lastSend = time.Now()

// sendMidiEvent sends a dummy MIDI event to conn,
// returning the expected event deadline.
func sendMidiEvent(
	evType midi.MidiEventType,
	channel,
	key,
	velocity uint8,
	conn chan midi.MidiEvent,
	duration time.Duration,
) time.Time {
	now := time.Now()
	timestamp := now.Sub(lastSend) / time.Millisecond
	if timestamp > 0xffffffff {
		panic("timestamp extrapolated an int32")
	}

	source := generateNoteEvent(evType, channel, key)

	deadline := time.Now().Add(duration)
	conn <- midi.MidiEvent{
		Source:    source[:],
		Timestamp: int32(timestamp),
		Type:      evType,
		Channel:   channel,
		Key:       key,
		Velocity:  velocity,
	}

	// Update the last send date.
	lastSend = now

	return deadline
}

// assertKeyEvent sends a MIDI event of the supplied on conn,
// checking that the keyCode was held down for duration every step.
// Since properly sync'ing this to the key event generator is somewhat tricky,
// this only fails if the key isn't released within graceTime of the duration.
func assertKeyEvent(
	t *testing.T,
	kc mockKeyController,
	keyCode int,
	evType midi.MidiEventType,
	channel,
	key,
	velocity uint8,
	conn chan midi.MidiEvent,
	duration,
	step,
	graceTime time.Duration,
) {
	start := time.Now()
	deadline := sendMidiEvent(evType, channel, key, velocity, conn, duration)

	// Wait a bit longer to be 100% sure the release event was handled.
	time.Sleep(step)

	count := 0
	for time.Now().Before(deadline) {
		assert(
			t,
			kc[keyCode] == true,
			"Key wasn't held down for the desired duration. Count=%d; Elapsed=%s",
			count,
			time.Now().Sub(start),
		)
		time.Sleep(step)
		count++
	}

	assert(t, count >= 1, "Key press wasn't checked even once")

	// Wait until the key is actually released
	// (assuming it may little longer than expected).
	for deadline := time.Now().Add(graceTime); kc[keyCode] && time.Now().Before(deadline); {
		time.Sleep(step)
	}

	// If the key wasn't actually released,
	// wait a bit longer so we may long when (and if) it was actually released.
	if kc[keyCode] != false {
		elapsed := time.Now().Sub(start)

		// Wait a bit more too see if the key would be release soonish.
		for i := 200; kc[keyCode] && i > 0; i-- {
			time.Sleep(time.Millisecond)
			i--
		}
		afterRetest := time.Now().Sub(start)

		var debug string
		if kc[keyCode] == false {
			debug = fmt.Sprintf("Key released after %s", afterRetest)
		} else {
			debug = fmt.Sprintf("Key wasn't released even after %s", afterRetest)
		}

		assert(
			t,
			false,
			"Key wasn't released. Elapsed=%s; %s",
			elapsed,
			debug,
		)
	}
}

func TestBasicPress(t *testing.T) {
	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := make(mockKeyController)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	const evType = midi.EventNoteOn
	const channel uint8 = 1
	const key uint8 = 2
	const keyCode int = 3
	const releaseTime = 10 * time.Millisecond
	const threshold = 30

	ke.RegisterBasicPressAction(
		evType,
		channel,
		key,
		keyCode,
		threshold,
		releaseTime,
	)

	// Test that sending a MIDI event different from the expected doesn't set the key.
	assert(t, kc[keyCode] == false, "Key was initially pressed")
	sendMidiEvent(evType, channel, key+1, 100, conn, releaseTime)
	time.Sleep(time.Millisecond)
	assert(t, kc[keyCode] == false, "Key was pressed by an invalid MIDI event")

	// Test that sending the event keeps the key pressed for the desired time.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		100,
		conn,
		releaseTime,
		time.Millisecond/2,
		time.Millisecond,
	)
}

func TestVelocityPress(t *testing.T) {
	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := make(mockKeyController)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	const evType = midi.EventNoteOn
	const channel uint8 = 1
	const key uint8 = 2
	const keyCode int = 3
	const minTime = 10 * time.Millisecond
	const maxTime = 100 * time.Millisecond
	const threshold = 0

	ke.RegisterVelocityAction(
		evType,
		channel,
		key,
		keyCode,
		threshold,
		minTime,
		maxTime,
	)

	// Test that sending a MIDI event different from the expected doesn't set the key.
	assert(t, kc[keyCode] == false, "Key was initially pressed")
	sendMidiEvent(evType, channel, key+1, 100, conn, minTime)
	time.Sleep(time.Millisecond)
	assert(t, kc[keyCode] == false, "Key was pressed by an invalid MIDI event")

	// Test that sending a quick MIDI event generates a quickly resolved key press.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		1,
		conn,
		minTime,
		time.Millisecond/2,
		time.Millisecond,
	)

	// Test that sending a long MIDI event generates a long-lasting key press.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		128,
		conn,
		maxTime,
		time.Millisecond/2,
		time.Millisecond,
	)
}

func TestHoldPress(t *testing.T) {
	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := make(mockKeyController)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	const evType = midi.EventNoteOn
	const channel uint8 = 1
	const key uint8 = 2
	const keyCode int = 3
	const shortRelease = 10 * time.Millisecond
	const maxDelayMs = 100
	const eventDelay = 95 * time.Millisecond
	const threshold = 30

	ke.RegisterHoldAction(
		evType,
		channel,
		key,
		keyCode,
		threshold,
		maxDelayMs,
		shortRelease,
	)

	// Test that sending a MIDI event different from the expected doesn't set the key.
	assert(t, kc[keyCode] == false, "Key was initially pressed")
	sendMidiEvent(evType, channel, key+1, 100, conn, shortRelease)
	time.Sleep(time.Millisecond)
	assert(t, kc[keyCode] == false, "Key was pressed by an invalid MIDI event")

	// Test that sending a quick MIDI event generates a quickly resolved key press.
	lastSend = time.Now().Add(-shortRelease - maxDelayMs*time.Millisecond)
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		100,
		conn,
		shortRelease,
		time.Millisecond/2,
		time.Millisecond,
	)

	// Test that sending repeated MIDI events generates a long-lasting key press.
	const count = 5
	const maxTime = eventDelay * count
	go func() {
		// Wait until the event was sent by assertKeyEvent()
		time.Sleep(time.Millisecond / 2)

		// Queue events roughly following the expected duration.
		for i := 0; i < count; i++ {
			sendMidiEvent(evType, channel, key, 100, conn, shortRelease)
			time.Sleep(eventDelay)
		}
	}()

	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		100,
		conn,
		maxTime,
		time.Millisecond/2,
		time.Millisecond*10,
	)
}

func TestTogglePress(t *testing.T) {
	conn := make(chan midi.MidiEvent, 1)
	defer close(conn)
	kc := make(mockKeyController)
	defer kc.Close()

	ke, err := NewKeyEvents(kc, conn, false)
	assert(t, err == nil, "Failed to start the key event generator")
	defer ke.Close()

	const evType = midi.EventNoteOn
	const channel uint8 = 1
	const key uint8 = 2
	const keyCode int = 3
	const shortRelease = 10 * time.Millisecond
	const threshold = 30
	const toggleThreshold = 80

	ke.RegisterToggleAction(
		evType,
		channel,
		key,
		keyCode,
		threshold,
		toggleThreshold,
		shortRelease,
	)

	// Test that sending a MIDI event different from the expected doesn't set the key.
	assert(t, kc[keyCode] == false, "Key was initially pressed")
	sendMidiEvent(evType, channel, key+1, 100, conn, shortRelease)
	time.Sleep(time.Millisecond)
	assert(t, kc[keyCode] == false, "Key was pressed by an invalid MIDI event")

	// Test that sending a quick MIDI event generates a quickly resolved key press.
	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		toggleThreshold-1,
		conn,
		shortRelease,
		time.Millisecond/2,
		time.Millisecond,
	)

	// Test that sending a long MIDI event holds the key down
	// until the event is sent again.
	const maxTime = time.Millisecond * 100
	go func() {
		// Queue an event to be sent after maxTime.
		time.Sleep(maxTime)
		sendMidiEvent(evType, channel, key, toggleThreshold, conn, shortRelease)
	}()

	assertKeyEvent(
		t,
		kc,
		keyCode,
		evType,
		channel,
		key,
		toggleThreshold+1,
		conn,
		maxTime,
		time.Millisecond/2,
		time.Millisecond*5,
	)
}
