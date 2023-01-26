# MIDI to key

## Quick start

Before starting, be sure to download `midicat`, since that's the only native driver in gitlab.com/gomidi/midi that works on Windows.
There's a download on the [driver's README](https://pkg.go.dev/gitlab.com/gomidi/midi/v2@v2.0.25/drivers/midicatdrv#section-readme),
but you should also be able to build your own from the `midicat` package: `gitlab.com/gomidi/midi/v2/tools/midicat/cmd/midicat`.

```bash
# Build the native application
go build .

# Cross-compile for Windows from Linux (or whatever)
GOOS=windows go build .
```

To run this on Linux, be sure to either copy `midicat` to a directory in your `$PATH`,
or alternatively simply add the directory `midicat` is in to the `$PATH`.
Assuming that both `midicat` and `midi-go-key` are in the current directory, you could:

```bash
export PATH=${PATH}:`pwd`

# Sending keys requires root
sudo ./midi-go-key
```

On Windows, simply have both binaries be on the same directory and it should work.

## Configuring inputs

This application allows mapping MIDI events into four different types of actions:

- Basic press: A simple key press followed by the key release shortly thereafter;
- Velocity-based press: A key press of variable hold type, calculated based on in velocity of the MIDI event;
- Toggle: Toggle a key between pressed and released whenever the MIDI event is generated. Additionally, if the event velocity is lower than a limit, a Basic Press is done instead;
- Repeated hold: Holds the key down while the MIDI event is repeated quickly.

These actions must be configured through the following script:

```
# Lines starting with '#' are comments (i.e., they are ignored by the application).
# NOTE: This application is quite limited and only accepts 'Note On' MIDI events, thus that's not configurable.

# Do a Basic press on MIDI event 41, holding 'A' down for 1000 millisecond (i.e., 1 second).
# The input is ignored if its velocity is less than 30 (considering that it goes from 0 to 128).
ch=9 ev=41 key=A thres=30 BASIC 1000

# Do a Velocity-based press on MIDI event 43, holding 'B' down for 10 an 1000 millisecond, based on the event velocity.
# The input is ignored if its velocity is less than 30 (considering that it goes from 0 to 128).
ch=9 ev=43 key=B thres=30 VELOCITY 10 1000

# Do a Toggle on MIDI event 44, toggle 'C' if the velocity is above 75 (considering that it goes from 0 to 128),
# otherwise holding it down for 10 milliseconds.
# The input is only ignored if its velocity is 0.
ch=9 ev=44 key=C thres=0 TOGGLE 75 10

# Do a Repeated hold on MIDI event 48 (i.e., hex 30), holding 'D' down if the event is repeated every 100 milliseconds.
# On the first (or only) event, the key is released after 10 milliseconds.
# The first input in a sequence is ignored if its velocity is less than 20 (considering that it goes from 0 to 128),
# but the following ones may be as light as you want.
ch=9 ev=0x30 key=D thres=20 REPEAT 100 10
```

Numbers may be written in any format, as long as they are properly prefixed.

## Testing

To run tests without installing `midicat`, specify the build tag `test`:

```bash
go test --tags=test ./...
```
