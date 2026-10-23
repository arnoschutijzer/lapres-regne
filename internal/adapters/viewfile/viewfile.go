package viewfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// File keeps the view the terminal last showed the world from as JSON at Path.
type File struct {
	Path string
}

// View is the camera's zoom, in steps in from the forest view, its yaw about the upright axis and
// its tilt above the horizon, in radians, and the name of the style the world is drawn in; a tilt
// of 0 and an empty style stand for the defaults.
type View struct {
	Zoom  int     `json:"zoom"`
	Yaw   float64 `json:"yaw"`
	Tilt  float64 `json:"tilt,omitempty"`
	Style string  `json:"style,omitempty"`
}

// Load returns the last saved view, or the forest view facing north when nothing was saved yet.
func (f File) Load() (View, error) {
	data, err := os.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return View{}, nil
	}
	if err != nil {
		return View{}, err
	}
	var view View
	if err := json.Unmarshal(data, &view); err != nil {
		return View{}, fmt.Errorf("read %s: %w", f.Path, err)
	}
	return view, nil
}

// Save writes a temporary file beside the view and renames it over the old one, so an
// interrupted save never leaves a half-written file behind.
func (f File) Save(view View) error {
	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(f.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".view-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), f.Path)
}
