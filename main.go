package main

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/LarsEckart/huey/config"
	"github.com/LarsEckart/huey/hue"
)

func main() {
	a := app.New()
	w := a.NewWindow("Huey")
	w.Resize(fyne.NewSize(300, 400))

	cfg, err := config.Load()
	if err != nil {
		w.SetContent(widget.NewLabel(fmt.Sprintf("Failed to load config: %v", err)))
		w.ShowAndRun()
		return
	}

	if !cfg.IsConfigured() {
		showSetup(w, a)
		return
	}

	client := hue.NewClient(cfg.BridgeIP, cfg.Username)
	showGroups(w, client)
	w.ShowAndRun()
}

func showSetup(w fyne.Window, a fyne.App) {
	bridgeEntry := widget.NewEntry()
	bridgeEntry.SetPlaceHolder("192.168.1.x")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("bridge username / API key")

	status := widget.NewLabel("")

	saveBtn := widget.NewButton("Connect", func() {
		ip := bridgeEntry.Text
		user := usernameEntry.Text
		if ip == "" || user == "" {
			status.SetText("Both fields are required.")
			return
		}
		cfg := &config.Config{BridgeIP: ip, Username: user}
		if err := cfg.Save(); err != nil {
			status.SetText(fmt.Sprintf("Failed to save: %v", err))
			return
		}
		client := hue.NewClient(ip, user)
		showGroups(w, client)
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Hue Bridge Setup"),
		widget.NewLabel("Bridge IP:"),
		bridgeEntry,
		widget.NewLabel("Username / API Key:"),
		usernameEntry,
		saveBtn,
		status,
	))
	w.ShowAndRun()
}

func showGroups(w fyne.Window, client *hue.Client) {
	content := container.NewVBox()
	statusLabel := widget.NewLabel("Loading...")
	content.Add(statusLabel)
	w.SetContent(container.NewVScroll(content))

	go loadGroups(w, client, content, statusLabel)
}

func loadGroups(w fyne.Window, client *hue.Client, content *fyne.Container, statusLabel *widget.Label) {
	groups, err := client.GetGroups()
	if err != nil {
		statusLabel.SetText(fmt.Sprintf("Error: %v", err))
		return
	}

	var filtered []hue.Group
	for _, g := range groups {
		if g.Type == "Room" || g.Type == "Zone" {
			filtered = append(filtered, g)
		}
	}

	content.RemoveAll()

	if len(filtered) == 0 {
		content.Add(widget.NewLabel("No rooms or zones found."))
		return
	}

	var mu sync.Mutex

	for _, g := range filtered {
		g := g
		label := widget.NewLabel(fmt.Sprintf("%s  (%s)", g.Name, g.Type))

		toggle := widget.NewCheck("", func(on bool) {
			mu.Lock()
			defer mu.Unlock()
			if err := client.SetGroupState(g.ID, hue.GroupAction{On: &on}); err != nil {
				dialog.ShowError(fmt.Errorf("failed to set %s: %w", g.Name, err), w)
			}
		})
		toggle.Checked = g.AnyOn

		row := container.NewHBox(toggle, label, layout.NewSpacer())
		content.Add(row)
	}

	refreshBtn := widget.NewButton("Refresh", func() {
		statusLabel := widget.NewLabel("Refreshing...")
		content.RemoveAll()
		content.Add(statusLabel)
		go loadGroups(w, client, content, statusLabel)
	})
	content.Add(layout.NewSpacer())
	content.Add(refreshBtn)
}
