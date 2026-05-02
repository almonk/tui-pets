package tui

import (
	"testing"
	"time"

	"tui-pets/internal/pets"
)

func testApp() *App {
	return &App{
		pet: pets.Pet{
			ID:          "boba",
			DisplayName: "Boba",
			FrameWidth:  192,
			FrameHeight: 208,
			Animations: map[string]pets.Animation{
				"idle":       {Name: "idle", Frames: []int{0}, FPS: 5, Loop: true},
				"wave":       {Name: "wave", Frames: []int{1}, FPS: 5, Loop: true},
				"move_left":  {Name: "move_left", Frames: []int{2}, FPS: 5, Loop: true},
				"move_right": {Name: "move_right", Frames: []int{3}, FPS: 5, Loop: true},
				"sip":        {Name: "sip", Frames: []int{4}, FPS: 5, Loop: true},
			},
		},
		currentAnimation: "idle",
		animationStarted: time.Now(),
		petCol:           8,
		petRow:           5,
		targetHeightPx:   75,
		running:          true,
	}
}

func TestPlainMouseMotionDoesNotMovePet(t *testing.T) {
	app := testApp()

	app.handleMouse([]string{"\x1b[<35;40;12M", "35", "40", "12", "M"})

	if app.petCol != 8 || app.petRow != 5 {
		t.Fatalf("plain mouse motion moved pet to col=%d row=%d", app.petCol, app.petRow)
	}
	if app.pointer.down {
		t.Fatal("plain mouse motion started a drag")
	}
}

func TestMousePressDoesNotMovePet(t *testing.T) {
	app := testApp()

	app.handleMouse([]string{"\x1b[<0;40;12M", "0", "40", "12", "M"})

	if app.petCol != 8 || app.petRow != 5 {
		t.Fatalf("mouse press moved pet to col=%d row=%d", app.petCol, app.petRow)
	}
	if !app.pointer.down {
		t.Fatal("mouse press did not start drag state")
	}
}

func TestDragMotionAfterPressMovesPet(t *testing.T) {
	app := testApp()

	app.handleMouse([]string{"\x1b[<0;40;12M", "0", "40", "12", "M"})
	size := app.imageSize(terminalGeometryFor(1))
	app.handleMouse([]string{"\x1b[<32;44;16M", "32", "44", "16", "M"})

	wantCol := 44 - size.columns/2
	wantRow := 16 - size.rows/2
	if app.petCol != wantCol || app.petRow != wantRow {
		t.Fatalf("drag moved pet to col=%d row=%d, want col=%d row=%d", app.petCol, app.petRow, wantCol, wantRow)
	}
}
