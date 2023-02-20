package key_events

import (
	"sync"
	"time"
)

// An action responsible for pressing/releasing a key.
type keyAction struct {
	// The key to be pressed/released.
	keyCode int
	// The internal key controller.
	kc KeyController
	// The timer used to release the generated key press.
	timer *time.Ticker
	// Synchronizes access to timer.
	mutex sync.Mutex
	// The action taken when the timer expires, if any.
	onTimeout timerAction
	// Channel used to signal that the action's timer should be stopped forever.
	stop chan struct{}
}

// newKeyAction creates a new keyAction, with its timer already configured (but stopped).
// onTimeout may be nil if no custom action is required after releasing the key.
func newKeyAction(
	keyCode int,
	kc KeyController,
	onTimeout timerAction,
) *keyAction {
	action := &keyAction{
		keyCode:        keyCode,
		kc:             kc,
		timer:          time.NewTicker(time.Second),
		onTimeout:      onTimeout,
		stop:           make(chan struct{}),
	}
	action.timer.Stop()
	go action.waitForTimedAction()

	return action
}

// Press presses the keyCode.
func (key *keyAction) Press() {
	key.kc.PressKeys(key.keyCode)
}

// release gets called automatically when key.timer expires.
func (key *keyAction) release() {
	key.kc.ReleaseKeys(key.keyCode)
	if key.onTimeout != nil {
		key.onTimeout()
	}
}

// UnqueueTimedAction releases the timeout from an action.
func (key *keyAction) UnqueueTimedAction() {
	key.mutex.Lock()
	key.timer.Stop()
	key.mutex.Unlock()
}

// QueueTimedAction queues an actions to be taken after timeout.
func (key *keyAction) QueueTimedAction(timeout time.Duration) {
	key.mutex.Lock()
	key.timer.Reset(timeout)
	key.mutex.Unlock()
}

// waitForTimedAction blocks until a timer event is generated,
// when it calls its onRelease action.
// After the actions is called, it stops it's timer.
func (key *keyAction) waitForTimedAction() {
	for {
		select {
		case <-key.stop:
			return
		case <-key.timer.C:
			// Do nothing and just exits the select.
		}
		key.release()

		key.mutex.Lock()
		key.timer.Stop()
		key.mutex.Unlock()
	}
}

// Close releases any resources associated with the MIDI action,
// and calls release.
func (key *keyAction) Close() error {
	if key.timer != nil {
		return nil
	}

	select {
	// If the ticker was already stopped, simply exit.
	case <-key.stop:
		return nil
	default:
	}

	// Stop and close the ticker, so it won't be triggered and
	// so the timer goroutine may exit.
	key.mutex.Lock()
	key.timer.Stop()
	key.mutex.Unlock()
	close(key.stop)

	// Make sure that the ticker is empty.
	for _ = range key.timer.C {
	}

	// Call the timeout action, just to be sure that the key is released.
	key.release()

	return nil
}
