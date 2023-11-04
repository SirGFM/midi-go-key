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
- Repeated hold: Holds the key down while the MIDI event is repeated quickly;
- Repeated Sequence: Use a MIDI event to press the current key, two MIDI events to move forward and backward in the sequence, and on MIDI event to reset back to the first key. This otherwise behaves like a Repeated hold.

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

# Do a Repeated hold on a Sequence of inputs on MIDI event 45 (i.e., hex 2d),
# holding the current key down if the event is repeated every 100 milliseconds.
# On the first (or only) event, the key is released after 10 milliseconds.
# The sequence starts on the named parameter 'key' (i.e., the up arrow key), and advances clockwise
# (thus, up/right, right, right/down, etc) whenever the MIDI event 43 (i.e., hex 2b) is received.
# If the MIDI event 48 (i.e., hex 30) is received instead, the sequence moves counter-clockwise.
# Additionally, MIDI event 38 (i.e., hex 26) can be used to reset back to the initial key (i.e., the up arrow key).
ch=9 ev=0x2d key=UP thres=20 REPEAT-SEQUENCE 100 10 0x30 0x2b 0x26 str=UP,RIGHT;RIGHT;RIGHT,DOWN;DOWN;DOWN,LEFT;LEFT;LEFT,UP

# If you need to dynamically change between a few sets of mappings,
# you can create a named set, which will contain every mapping within it.
# By default, these mappings won't be used, so you must define which set is in use,
# as well as which MIDI event changes the sets.
#
# Note that the USE_MAPPING action should be defined outside of any mapping,
# otherwise this action could be overwritten in another mapping.
#
# Also, if a key is both in a named set and in the default, unnamed set,
# the action in the default, unnamed set takes precedence.

# Define SET_A as the initially active set,
# using MIDI event 40 (i.e., hex 28) to advance to the next set in the sequence,
# separated by commas.
# Because of how the parser was implemented, a key must be defined.
# However since this value isn't used, the special name 'NONE'
# can be used (which doesn't map to any key).
ch=9 ev=0x28 key=NONE thres=20 USE-MAPPING str=SET_A,SET_B,SET_C

# Create a new mapping set called SET_A,
# with a Basic action on MIDI event 50 to key 'Z'.
#
# Because of how the parser was implemented,
# every argument must be supplied with some dummy value.
ch=0 ev=0 key=NONE thres=0 NEW-MAPPING str=SET_A

ch=9 ev=50 key=Z thres=30 BASIC 1000

# Create a new mapping set called SET_B,
# with a Basic action on MIDI event 50 to key 'X'.
ch=0 ev=0 key=NONE thres=0 NEW-MAPPING str=SET_B

ch=9 ev=50 key=X thres=30 BASIC 1000

# Create a new mapping set called SET_C,
# with a Basic action on MIDI event 50 to key 'C'.
ch=0 ev=0 key=NONE thres=0 NEW-MAPPING str=SET_C

ch=9 ev=50 key=C thres=30 BASIC 1000
```

Numbers may be written in any format, as long as they are properly prefixed.

## Testing

To run tests without installing `midicat`, specify the build tag `test`:

```bash
go test --tags=test ./...
```
