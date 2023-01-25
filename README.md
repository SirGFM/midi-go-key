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
