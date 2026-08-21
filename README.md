# Huey Win

A minimal Windows GUI for controlling Philips Hue rooms and zones.

Built with [Fyne](https://fyne.io/) and the [huey](https://github.com/LarsEckart/huey) library.

## Features

- Discover and pair with your Hue bridge
- List all rooms and zones
- Toggle lights on/off with a single click

## Installation

Download `huey-win.exe` from the [Releases](https://github.com/LarsEckart/huey-win/releases) page and run it. No installation required.

## Usage

1. Run `huey-win.exe`
2. On first launch, Huey Win automatically looks for your Hue bridge via `https://discovery.meethue.com/`
3. If multiple bridges are found, choose one; if none are found, enter the bridge IP manually
4. Press the physical button on your bridge, then click **Pair**
5. Your rooms and zones appear — tap any row to toggle lights

Configuration is stored in `~/.config/huey/config.json`.

## Building from source

```
go build -ldflags="-H windowsgui" -o huey-win.exe .
```

Requires Go 1.27+ and a C compiler (CGO is needed for Fyne).

## Privacy policy

This program communicates with the Philips Hue bridge on your local network. During setup, it may also call Philips Hue's discovery service at `https://discovery.meethue.com/` to find your bridge IP address.

## Code signing policy

Free code signing provided by [SignPath.io](https://about.signpath.io), certificate by [SignPath Foundation](https://signpath.org).

### Team roles

- Committers and reviewers: [LarsEckart](https://github.com/LarsEckart)
- Approvers: [LarsEckart](https://github.com/LarsEckart)

## License

[MIT](LICENSE)
