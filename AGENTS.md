## huey-win

A minimal Fyne GUI for controlling Philips Hue rooms/zones on Windows. Single-file app (`main.go`).

## Commands

- Build: `go build -o huey-win.exe .` (Windows: add `-ldflags="-H windowsgui"`)
- Build (macOS dev): `go build -o huey-win .`
- Test: `go test ./...`
- Cross-compile for Windows: `GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-H windowsgui" -o huey-win.exe .`

## Architecture

All code lives in `main.go`. Bridge communication and config use packages from the `github.com/LarsEckart/huey` module (`hue/client.go` for HTTP calls, `config/config.go` for `~/.config/huey/config.json`). UI is built with **Fyne v2**. No console output — all errors must be shown via Fyne widgets/dialogs (no `log.Fatal` or `fmt.Print`).

## Releasing

Push a version tag to trigger a release: `git tag v0.1.0 && git push origin v0.1.0`. The GitHub Actions workflow will build → sign (via SignPath) → create a GitHub Release with the signed `huey-win.exe` attached.

## Code Style

- Go standard formatting (`gofmt`). Imports: stdlib, blank line, external, blank line, internal.
- Errors in UI callbacks: use `dialog.ShowError(...)` or update a `widget.Label`. Never write to stdout/stderr.
- Bridge calls that block must run in a goroutine (`go func() { ... }()`); protect shared state with `sync.Mutex`.
- Keep it simple — this is intentionally a single-file app. No unnecessary abstractions.
