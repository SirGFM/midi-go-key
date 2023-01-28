package key_handler

import (
	"runtime"
	"time"

	"github.com/SirGFM/midi-go-key/err_wrap"
	"github.com/micmonay/keybd_event"
)

type keyHandler struct {
	// The internal key events generator.
	kb *keybd_event.KeyBonding
}

// Configures a new keyboard handler.
func New() (*keyHandler, error) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return nil, err_wrap.Wrap(err, ErrGetKeyGenerator)
	}

	// For linux, it is very important to wait 2 seconds
	if runtime.GOOS == "linux" {
		time.Sleep(2 * time.Second)
	}

	return &keyHandler{
		kb: &kb,
	}, nil
}

// Releases every resource associated with the key controller.
func (ctx *keyHandler) Close() error {
	return nil
}

// PressKeys presses the requested keys, by their keycode.
func (ctx *keyHandler) PressKeys(keyCodes ...int) {
	ctx.kb.SetKeys(keyCodes...)
	ctx.kb.Press()
}

// ReleaseKeys releases the requested keys, by their keycode.
func (ctx *keyHandler) ReleaseKeys(keyCodes ...int) {
	ctx.kb.SetKeys(keyCodes...)
	ctx.kb.Release()
}
