package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Intensity levels, in menu order.
const (
	IntensityOff = iota
	IntensityLight
	IntensitySteady
	IntensityHeavy
	IntensityBlizzard
	IntensityVariable
)

// Prefs are the user's settings, kept in %APPDATA%\SnowForWindows\settings.json.
type Prefs struct {
	Intensity    int  `json:"intensity"`
	Accumulate   bool `json:"accumulate"`
	Melt         bool `json:"melt"`
	OnWindows    bool `json:"snowOnWindows"`
	InScreenshot bool `json:"showInScreenshots"`
	Smooth       bool `json:"smooth60fps"`
	Welcomed     bool `json:"welcomed"`
}

// DefaultPrefs is what a first launch uses.
func DefaultPrefs() Prefs {
	return Prefs{Intensity: IntensitySteady, Accumulate: true, Melt: true, OnWindows: true}
}

// Level is how much snow an intensity setting makes, relative to a steady fall, and whether it varies.
func Level(intensity int) (float32, bool) {
	switch intensity {
	case IntensityLight:
		return 0.45, false
	case IntensitySteady:
		return 1, false
	case IntensityHeavy:
		return 1.9, false
	case IntensityBlizzard:
		return 3.2, false
	case IntensityVariable:
		return 2.2, true
	}
	return 0, false
}

func prefsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SnowForWindows", "settings.json")
}

// LoadPrefs reads the settings, or the defaults if there are none yet.
func LoadPrefs() Prefs {
	p := DefaultPrefs()
	path := prefsPath()
	if path == "" {
		return p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, &p)
	if p.Intensity < IntensityOff || p.Intensity > IntensityVariable {
		p.Intensity = IntensitySteady
	}
	return p
}

// Save writes the settings, quietly giving up if the folder can't be written.
func (p Prefs) Save() {
	path := prefsPath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, _ := json.MarshalIndent(p, "", "  ")
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}
