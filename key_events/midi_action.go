package key_events

import (
	"sync"
	"time"

	"github.com/SirGFM/midi-go-key/midi"
)

// An action taken in response to a MIDI event.
type midiAction struct {
	// The action taken (e.g., generate a key press).
	action func(midi.MidiEvent)
	// The timer used to release the generated key press.
	timer *time.Ticker
	// The action taken when the timer expires.
	onTimeout timerAction
	// Synchronizes access to timer.
	mutex sync.Mutex
	// Channel used to signal that the action's timer should be stopped forever.
	stop chan struct{}
}

// newMidiAction creates a new midiAction, with its timer already configured (but stopped).
func newMidiAction() *midiAction {
	handler := &midiAction{
		timer: time.NewTicker(time.Second),
		stop:  make(chan struct{}),
	}
	handler.timer.Stop()
	go handler.WaitForTimedAction()

	return handler
}

// QueueTimedAction queues an actions to be taken after timeout.
func (ma *midiAction) QueueTimedAction(timeout time.Duration) {
	ma.mutex.Lock()
	ma.timer.Reset(timeout)
	ma.mutex.Unlock()
}

// WaitForTimedAction blocks until a timer event is generated,
// when it calls its onTimeout action.
// After the actions is called, it stops it's timer.
func (ma *midiAction) WaitForTimedAction() {
	for {
		select {
		case <-ma.stop:
			return
		case <-ma.timer.C:
			// Do nothing and just exits the select.
		}
		ma.onTimeout()

		ma.mutex.Lock()
		ma.timer.Stop()
		ma.mutex.Unlock()
	}
}

// Close releases any resources associated with the MIDI action,
// and calls onTimeout.
func (ma *midiAction) Close() error {
	if ma.timer != nil {
		return nil
	}

	select {
	// If the ticker was already stopped, simply exit.
	case <-ma.stop:
		return nil
	default:
	}

	// Stop and close the ticker, so it won't be triggered and
	// so the timer goroutine may exit.
	ma.mutex.Lock()
	ma.timer.Stop()
	ma.mutex.Unlock()
	close(ma.stop)

	// Make sure that the ticker is empty.
	for _ = range ma.timer.C {
	}

	// Call the timeout action, just to be sure that the key is released.
	if ma.onTimeout != nil {
		ma.onTimeout()
	}

	return nil
}
