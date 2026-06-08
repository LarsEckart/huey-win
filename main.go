package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

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
	a.SetIcon(appIcon)
	w := a.NewWindow("Huey")
	w.Resize(fyne.NewSize(300, 400))

	cfg, err := config.Load()
	if err != nil {
		w.SetContent(widget.NewLabel(fmt.Sprintf("Failed to load config: %v", err)))
		w.ShowAndRun()
		return
	}

	if !cfg.IsConfigured() {
		showSetup(w, a, cfg)
		return
	}

	client := hue.NewClient(cfg.BridgeIP, cfg.Username)
	showTabs(w, client)
	w.ShowAndRun()
}

func showSetup(w fyne.Window, a fyne.App, cfg *config.Config) {
	bridgeEntry := widget.NewEntry()
	bridgeEntry.SetPlaceHolder("192.168.1.x")

	status := widget.NewLabel("Looking for Hue bridges...")
	bridgeOptions := make(map[string]string)
	bridgeSelect := widget.NewSelect(nil, func(selected string) {
		if ip := bridgeOptions[selected]; ip != "" {
			bridgeEntry.SetText(ip)
		}
	})
	bridgeSelect.PlaceHolder = "Choose a detected bridge"
	bridgeSelect.Hide()

	pairLabel := "Pair"
	setupHint := "Press the button on your bridge, then click Pair."
	if cfg.Username != "" {
		pairLabel = "Use Saved Pairing"
		setupHint = "Choose or enter your bridge IP to use the saved pairing."
	}

	var pairBtn *widget.Button
	pairBtn = widget.NewButton(pairLabel, func() {
		ip := strings.TrimSpace(bridgeEntry.Text)
		if ip == "" {
			status.SetText("Enter the bridge IP address.")
			return
		}
		pairBtn.Disable()

		go func() {
			if cfg.Username != "" {
				if err := useSavedPairing(w, cfg, ip); err != nil {
					status.SetText(fmt.Sprintf("Failed to save config: %v", err))
					pairBtn.Enable()
				}
				return
			}

			status.SetText("Press the button on your Hue bridge, then wait...")
			client := hue.NewClient(ip, "")
			username, err := client.Register("huey-win#pc")
			if err != nil {
				status.SetText(fmt.Sprintf("Error: %v", err))
				pairBtn.Enable()
				return
			}
			cfg := &config.Config{BridgeIP: ip, Username: username}
			if err := cfg.Save(); err != nil {
				status.SetText(fmt.Sprintf("Failed to save config: %v", err))
				pairBtn.Enable()
				return
			}
			authedClient := hue.NewClient(ip, username)
			showTabs(w, authedClient)
		}()
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Hue Bridge Setup"),
		widget.NewLabel("Bridge IP:"),
		bridgeSelect,
		bridgeEntry,
		widget.NewLabel(""),
		widget.NewLabel(setupHint),
		pairBtn,
		status,
	))

	go discoverSetupBridges(w, cfg, bridgeEntry, bridgeSelect, bridgeOptions, status)
	w.ShowAndRun()
}

func useSavedPairing(w fyne.Window, cfg *config.Config, ip string) error {
	cfg.BridgeIP = ip
	if err := cfg.Save(); err != nil {
		return err
	}

	client := hue.NewClient(cfg.BridgeIP, cfg.Username)
	showTabs(w, client)
	return nil
}

func discoverSetupBridges(
	w fyne.Window,
	cfg *config.Config,
	bridgeEntry *widget.Entry,
	bridgeSelect *widget.Select,
	bridgeOptions map[string]string,
	status *widget.Label,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bridges, err := discoverBridges(ctx)
	if err != nil {
		status.SetText("Automatic discovery failed. Enter your bridge IP manually.")
		return
	}

	switch len(bridges) {
	case 0:
		status.SetText("No bridge found automatically. Enter your bridge IP manually.")
	case 1:
		bridgeEntry.SetText(bridges[0].InternalIPAddress)
		if cfg.Username != "" {
			if err := useSavedPairing(w, cfg, bridges[0].InternalIPAddress); err != nil {
				status.SetText(fmt.Sprintf("Failed to save config: %v", err))
			}
			return
		}
		status.SetText(fmt.Sprintf("Found bridge at %s. Press the bridge button, then click Pair.", bridges[0].InternalIPAddress))
	default:
		labels := make([]string, 0, len(bridges))
		for _, bridge := range bridges {
			label := bridge.DisplayName()
			bridgeOptions[label] = bridge.InternalIPAddress
			labels = append(labels, label)
		}
		bridgeSelect.Options = labels
		bridgeSelect.SetSelected(labels[0])
		bridgeSelect.Show()
		bridgeEntry.SetText(bridges[0].InternalIPAddress)
		if cfg.Username != "" {
			status.SetText("Choose your bridge, then click Use Saved Pairing.")
			return
		}
		status.SetText("Choose your bridge, press its button, then click Pair.")
	}
}

type discoveredBridge struct {
	ID                string `json:"id"`
	InternalIPAddress string `json:"internalipaddress"`
}

func (bridge discoveredBridge) DisplayName() string {
	if bridge.ID == "" {
		return bridge.InternalIPAddress
	}
	return fmt.Sprintf("%s (%s)", bridge.InternalIPAddress, bridge.ID)
}

func discoverBridges(ctx context.Context) ([]discoveredBridge, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discovery.meethue.com/", nil)
	if err != nil {
		return nil, fmt.Errorf("create discovery request: %w", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get discovery service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery service returned %s", resp.Status)
	}

	var bridges []discoveredBridge
	if err := json.NewDecoder(resp.Body).Decode(&bridges); err != nil {
		return nil, fmt.Errorf("decode discovery response: %w", err)
	}

	return normalizeDiscoveredBridges(bridges), nil
}

func normalizeDiscoveredBridges(bridges []discoveredBridge) []discoveredBridge {
	unique := make(map[string]discoveredBridge, len(bridges))
	for _, bridge := range bridges {
		bridge.ID = strings.TrimSpace(bridge.ID)
		bridge.InternalIPAddress = strings.TrimSpace(bridge.InternalIPAddress)
		if bridge.InternalIPAddress == "" {
			continue
		}
		key := cmp.Or(bridge.ID, bridge.InternalIPAddress)
		unique[key] = bridge
	}

	normalized := make([]discoveredBridge, 0, len(unique))
	for _, bridge := range unique {
		normalized = append(normalized, bridge)
	}

	slices.SortFunc(normalized, func(a, b discoveredBridge) int {
		if idOrder := cmp.Compare(a.ID, b.ID); idOrder != 0 {
			return idOrder
		}
		return cmp.Compare(a.InternalIPAddress, b.InternalIPAddress)
	})

	return normalized
}

func showTabs(w fyne.Window, client *hue.Client) {
	groupsContent := container.NewVBox()
	groupsStatus := widget.NewLabel("Loading...")
	groupsContent.Add(groupsStatus)

	lightsContent := container.NewVBox()
	lightsStatus := widget.NewLabel("Loading...")
	lightsContent.Add(lightsStatus)

	tabs := container.NewAppTabs(
		container.NewTabItem("Rooms", container.NewVScroll(groupsContent)),
		container.NewTabItem("Lights", container.NewVScroll(lightsContent)),
	)

	w.SetContent(tabs)

	go loadGroups(w, client, groupsContent, groupsStatus)
	go loadLights(w, client, lightsContent, lightsStatus)
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

		rowContent := container.NewHBox(toggle, label, layout.NewSpacer())
		tappableRow := newTappableRow(rowContent, func() {
			toggle.SetChecked(!toggle.Checked)
		})
		content.Add(tappableRow)
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

func loadLights(w fyne.Window, client *hue.Client, content *fyne.Container, statusLabel *widget.Label) {
	lights, err := client.GetLights()
	if err != nil {
		statusLabel.SetText(fmt.Sprintf("Error: %v", err))
		return
	}

	content.RemoveAll()

	if len(lights) == 0 {
		content.Add(widget.NewLabel("No lights found."))
		return
	}

	var mu sync.Mutex

	for _, l := range lights {
		l := l
		label := widget.NewLabel(fmt.Sprintf("%s  (%s)", l.Name, l.Type))

		toggle := widget.NewCheck("", func(on bool) {
			mu.Lock()
			defer mu.Unlock()
			if err := client.SetLightState(l.ID, hue.LightState{On: &on}); err != nil {
				dialog.ShowError(fmt.Errorf("failed to set %s: %w", l.Name, err), w)
			}
		})
		toggle.Checked = l.On

		bri := l.Brightness
		slider := widget.NewSlider(0, 254)
		slider.Value = float64(bri)
		slider.OnChanged = func(val float64) {
			mu.Lock()
			defer mu.Unlock()
			b := int(val)
			go func() {
				if err := client.SetLightState(l.ID, hue.LightState{Brightness: &b}); err != nil {
					dialog.ShowError(fmt.Errorf("failed to set brightness for %s: %w", l.Name, err), w)
				}
			}()
		}

		tappableRow := newTappableRow(container.NewHBox(toggle, label, layout.NewSpacer()), func() {
			toggle.SetChecked(!toggle.Checked)
		})
		content.Add(container.NewVBox(tappableRow, slider))
	}

	refreshBtn := widget.NewButton("Refresh", func() {
		statusLabel := widget.NewLabel("Refreshing...")
		content.RemoveAll()
		content.Add(statusLabel)
		go loadLights(w, client, content, statusLabel)
	})
	content.Add(layout.NewSpacer())
	content.Add(refreshBtn)
}

// tappableRow wraps any content and makes the entire area tappable.
type tappableRow struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newTappableRow(content fyne.CanvasObject, onTap func()) *tappableRow {
	t := &tappableRow{content: content, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *tappableRow) Tapped(_ *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}
