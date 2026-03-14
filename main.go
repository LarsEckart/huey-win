package main

import (
	"fmt"
	"log"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/LarsEckart/huey/config"
	"github.com/LarsEckart/huey/hue"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if !cfg.IsConfigured() {
		log.Fatal("Not configured. Run huey first to set up bridge IP and username.")
	}

	client := hue.NewClient(cfg.BridgeIP, cfg.Username)

	a := app.New()
	w := a.NewWindow("Huey")
	w.Resize(fyne.NewSize(300, 400))

	content := container.NewVBox()
	statusLabel := widget.NewLabel("Loading...")
	content.Add(statusLabel)

	w.SetContent(container.NewVScroll(content))

	go loadGroups(client, content, statusLabel)

	w.ShowAndRun()
}

func loadGroups(client *hue.Client, content *fyne.Container, statusLabel *widget.Label) {
	groups, err := client.GetGroups()
	if err != nil {
		statusLabel.SetText(fmt.Sprintf("Error: %v", err))
		return
	}

	// Filter to only rooms and zones.
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
				log.Printf("Failed to set %s: %v", g.Name, err)
			}
		})
		toggle.Checked = g.AnyOn

		row := container.NewHBox(toggle, label, layout.NewSpacer())
		content.Add(row)
	}

	// Add a refresh button at the bottom.
	refreshBtn := widget.NewButton("Refresh", func() {
		statusLabel := widget.NewLabel("Refreshing...")
		content.RemoveAll()
		content.Add(statusLabel)
		go loadGroups(client, content, statusLabel)
	})
	content.Add(layout.NewSpacer())
	content.Add(refreshBtn)
}
