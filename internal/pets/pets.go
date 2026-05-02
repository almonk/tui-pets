package pets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Animation struct {
	Name     string
	Frames   []int
	FPS      float64
	Loop     bool
	Fallback string
}

type Pet struct {
	ID              string
	DisplayName     string
	Description     string
	SpritesheetPath string
	FrameWidth      int
	FrameHeight     int
	Columns         int
	Rows            int
	Animations      map[string]Animation
}

func (p Pet) FrameCount() int {
	return p.Columns * p.Rows
}

type petFile struct {
	ID              string                   `json:"id"`
	DisplayName     string                   `json:"displayName"`
	Description     string                   `json:"description"`
	SpritesheetPath string                   `json:"spritesheetPath"`
	Frame           frameSpec                `json:"frame"`
	Animations      map[string]animationSpec `json:"animations"`
}

type frameSpec struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Columns int `json:"columns"`
	Rows    int `json:"rows"`
}

type animationSpec struct {
	Frames   []int    `json:"frames"`
	FPS      *float64 `json:"fps"`
	Loop     *bool    `json:"loop"`
	Fallback string   `json:"fallback"`
}

func Load(id string) (Pet, error) {
	petDir, err := resolvePetDir(id)
	if err != nil {
		return Pet{}, err
	}

	configPath := filepath.Join(petDir, "pet.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return Pet{}, fmt.Errorf("read %s: %w", configPath, err)
	}

	var file petFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return Pet{}, fmt.Errorf("parse %s: %w", configPath, err)
	}

	if file.ID == "" {
		file.ID = filepath.Base(petDir)
	}
	if file.DisplayName == "" {
		file.DisplayName = file.ID
	}
	if file.SpritesheetPath == "" {
		file.SpritesheetPath = "spritesheet.webp"
	}

	frame := file.Frame
	if frame.Width == 0 || frame.Height == 0 || frame.Columns == 0 || frame.Rows == 0 {
		frame = defaultFrame()
	}

	spritesheetPath := file.SpritesheetPath
	if !filepath.IsAbs(spritesheetPath) {
		spritesheetPath = filepath.Join(petDir, spritesheetPath)
	}
	if _, err := os.Stat(spritesheetPath); err != nil {
		return Pet{}, fmt.Errorf("missing spritesheet %s: %w", spritesheetPath, err)
	}

	animations := loadAnimations(file.Animations)

	return Pet{
		ID:              file.ID,
		DisplayName:     file.DisplayName,
		Description:     file.Description,
		SpritesheetPath: spritesheetPath,
		FrameWidth:      frame.Width,
		FrameHeight:     frame.Height,
		Columns:         frame.Columns,
		Rows:            frame.Rows,
		Animations:      animations,
	}, nil
}

func resolvePetDir(value string) (string, error) {
	if pathLike(value) {
		path, err := expandPath(value)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("pet path %s: %w", path, err)
		}
		if !info.IsDir() {
			path = filepath.Dir(path)
		}
		return filepath.Abs(path)
	}

	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "pets", value), nil
}

func pathLike(value string) bool {
	return value == "." ||
		value == ".." ||
		strings.HasPrefix(value, "~/") ||
		strings.HasPrefix(value, "../") ||
		strings.HasPrefix(value, "./") ||
		filepath.IsAbs(value) ||
		strings.ContainsRune(value, filepath.Separator)
}

func expandPath(value string) (string, error) {
	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if value == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(value, "~/")), nil
	}
	return value, nil
}

func defaultFrame() frameSpec {
	return frameSpec{
		Width:   192,
		Height:  208,
		Columns: 8,
		Rows:    9,
	}
}

func loadAnimations(specs map[string]animationSpec) map[string]Animation {
	animations := defaultAnimations()
	if len(specs) == 0 {
		return animations
	}

	for name, spec := range specs {
		if len(spec.Frames) == 0 {
			continue
		}

		fps := 8.0
		if spec.FPS != nil {
			fps = *spec.FPS
		}
		if fps <= 0 {
			fps = 8.0
		}

		loop := true
		if spec.Loop != nil {
			loop = *spec.Loop
		}

		fallback := spec.Fallback
		if fallback == "" {
			fallback = "idle"
		}

		animations[name] = Animation{
			Name:     name,
			Frames:   spec.Frames,
			FPS:      fps,
			Loop:     loop,
			Fallback: fallback,
		}
	}

	if _, ok := animations["idle"]; !ok {
		animations["idle"] = defaultAnimations()["idle"]
	}

	return animations
}

func defaultAnimations() map[string]Animation {
	return map[string]Animation{
		"idle":       {Name: "idle", Frames: []int{0, 1, 2, 3, 4, 5}, FPS: 5, Loop: true, Fallback: "idle"},
		"move_left":  {Name: "move_left", Frames: []int{8, 9, 10, 11, 12, 13, 14, 15}, FPS: 10, Loop: true, Fallback: "idle"},
		"move_right": {Name: "move_right", Frames: []int{16, 17, 18, 19, 20, 21, 22, 23}, FPS: 10, Loop: true, Fallback: "idle"},
		"wave":       {Name: "wave", Frames: []int{24, 25, 26, 27}, FPS: 7, Loop: false, Fallback: "idle"},
		"sit":        {Name: "sit", Frames: []int{32, 33, 34, 35, 36}, FPS: 6, Loop: true, Fallback: "idle"},
		"sad":        {Name: "sad", Frames: []int{40, 41, 42, 43, 44, 45, 46}, FPS: 6, Loop: true, Fallback: "idle"},
		"sleep":      {Name: "sleep", Frames: []int{43, 44, 47}, FPS: 3, Loop: true, Fallback: "idle"},
		"sip":        {Name: "sip", Frames: []int{48, 49, 50, 51, 52, 53}, FPS: 8, Loop: false, Fallback: "idle"},
		"bounce":     {Name: "bounce", Frames: []int{56, 57, 58, 59, 60, 61}, FPS: 9, Loop: false, Fallback: "idle"},
		"grumpy":     {Name: "grumpy", Frames: []int{64, 65, 66, 67, 68, 69}, FPS: 6, Loop: false, Fallback: "idle"},
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "pets")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root containing pets/")
		}
	}
}
