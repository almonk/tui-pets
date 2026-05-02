package tui

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"tui-pets/internal/kitty"
	"tui-pets/internal/pets"
)

var sgrMouseRE = regexp.MustCompile(`^\x1b\[<(\d+);(\d+);(\d+)([Mm])`)

type pointer struct {
	x    int
	y    int
	down bool
}

type App struct {
	pet              pets.Pet
	frameDir         string
	frames           []string
	currentAnimation string
	animationStarted time.Time
	petCol           int
	petRow           int
	targetHeightPx   int
	pointer          pointer
	inputBuffer      string
	running          bool
	message          string
}

func New(pet pets.Pet) *App {
	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		cacheDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}

	return &App{
		pet:              pet,
		frameDir:         filepath.Join(cacheDir, "tui-pets", pet.ID, "frames"),
		currentAnimation: "idle",
		animationStarted: time.Now(),
		petCol:           8,
		petRow:           5,
		targetHeightPx:   75,
		running:          true,
	}
}

func (a *App) Frames() []string {
	return a.frames
}

type terminalGeometry struct {
	columns      int
	rows         int
	cellWidthPx  float64
	cellHeightPx float64
}

type imageSize struct {
	columns  int
	rows     int
	heightPx int
}

func terminalGeometryFor(fd int) terminalGeometry {
	columns, rows, err := term.GetSize(fd)
	if err != nil {
		columns, rows = 100, 32
	}

	cellWidthPx := 0.0
	cellHeightPx := 0.0
	if ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ); err == nil {
		if ws.Xpixel > 0 && ws.Ypixel > 0 && ws.Col > 0 && ws.Row > 0 {
			cellWidthPx = float64(ws.Xpixel) / float64(ws.Col)
			cellHeightPx = float64(ws.Ypixel) / float64(ws.Row)
		}
	}

	return terminalGeometry{
		columns:      columns,
		rows:         rows,
		cellWidthPx:  cellWidthPx,
		cellHeightPx: cellHeightPx,
	}
}

func (a *App) imageSize(geometry terminalGeometry) imageSize {
	if geometry.cellWidthPx > 0 && geometry.cellHeightPx > 0 {
		rows := max(1, int(math.Round(float64(a.targetHeightPx)/geometry.cellHeightPx)))
		heightPx := int(math.Round(float64(rows) * geometry.cellHeightPx))
		widthPx := float64(heightPx) * float64(a.pet.FrameWidth) / float64(a.pet.FrameHeight)
		columns := max(1, int(math.Round(widthPx/geometry.cellWidthPx)))
		return imageSize{columns: columns, rows: rows, heightPx: heightPx}
	}

	rows := max(1, int(math.Round(float64(a.targetHeightPx)/15.0)))
	columns := max(1, int(math.Round(float64(rows)/(float64(a.pet.FrameHeight)/float64(a.pet.FrameWidth)*0.52))))
	return imageSize{columns: columns, rows: rows, heightPx: a.targetHeightPx}
}

func (a *App) PrepareFrames() error {
	if _, err := exec.LookPath("magick"); err != nil {
		return fmt.Errorf("ImageMagick `magick` was not found; install ImageMagick to slice spritesheets")
	}

	if err := os.MkdirAll(a.frameDir, 0o755); err != nil {
		return err
	}

	expected := make([]string, 0, a.pet.FrameCount())
	for i := 0; i < a.pet.FrameCount(); i++ {
		expected = append(expected, filepath.Join(a.frameDir, fmt.Sprintf("frame_%03d.png", i)))
	}

	complete := true
	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			complete = false
			break
		}
	}

	if !complete {
		stale, _ := filepath.Glob(filepath.Join(a.frameDir, "frame_*.png"))
		for _, path := range stale {
			_ = os.Remove(path)
		}

		output := filepath.Join(a.frameDir, "frame_%03d.png")
		cmd := exec.Command(
			"magick",
			a.pet.SpritesheetPath,
			"-crop",
			fmt.Sprintf("%dx%d", a.pet.FrameWidth, a.pet.FrameHeight),
			"+repage",
			output,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("slice spritesheet: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
	}

	a.frames = expected
	return nil
}

func (a *App) Run() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	input := make(chan []byte, 16)
	done := make(chan struct{})
	go readInput(os.Stdin, input, done)

	fmt.Print(kitty.AltScreenOn() + kitty.MouseOn())
	a.centerPet()

	defer func() {
		close(done)
		fmt.Print(kitty.MouseOff() + kitty.AltScreenOff())
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	return a.loop(input)
}

func readInput(reader io.Reader, out chan<- []byte, done <-chan struct{}) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			select {
			case out <- chunk:
			case <-done:
				return
			}
		}
		if err != nil {
			return
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func (a *App) centerPet() {
	geometry := terminalGeometryFor(int(os.Stdout.Fd()))
	size := a.imageSize(geometry)
	a.petCol = max(1, (geometry.columns-size.columns)/2)
	a.petRow = max(3, (geometry.rows-size.rows)/2)
}

func (a *App) loop(input <-chan []byte) error {
	nextFrame := time.Now()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()

	for a.running {
		select {
		case chunk := <-input:
			a.inputBuffer += string(chunk)
			a.consumeInputBuffer()
		case now := <-tick.C:
			if !now.Before(nextFrame) {
				a.render(now)
				fps := max(1, int(a.animation().FPS))
				nextFrame = now.Add(time.Second / time.Duration(fps))
			}
		}
	}

	return nil
}

func (a *App) animation() pets.Animation {
	if animation, ok := a.pet.Animations[a.currentAnimation]; ok {
		return animation
	}
	return a.pet.Animations["idle"]
}

func (a *App) setAnimation(name string) {
	if name == a.currentAnimation {
		return
	}
	if _, ok := a.pet.Animations[name]; !ok {
		return
	}
	a.currentAnimation = name
	a.animationStarted = time.Now()
}

func (a *App) framePath(now time.Time) string {
	animation := a.animation()
	if len(animation.Frames) == 0 {
		animation = a.pet.Animations["idle"]
	}

	elapsed := now.Sub(a.animationStarted).Seconds()
	index := int(elapsed * animation.FPS)

	var frameIndex int
	if animation.Loop {
		frameIndex = animation.Frames[index%len(animation.Frames)]
	} else if index >= len(animation.Frames) {
		a.setAnimation(animation.Fallback)
		frameIndex = a.animation().Frames[0]
	} else {
		frameIndex = animation.Frames[index]
	}

	return a.frames[frameIndex]
}

func (a *App) render(now time.Time) {
	geometry := terminalGeometryFor(int(os.Stdout.Fd()))
	size := a.imageSize(geometry)

	a.petCol = min(max(1, a.petCol), max(1, geometry.columns-size.columns))
	a.petRow = min(max(3, a.petRow), max(3, geometry.rows-size.rows))

	lines := []string{
		fmt.Sprintf("%s  %s  ~%dpx high  %dc x %dr", a.pet.DisplayName, a.currentAnimation, size.heightPx, size.columns, size.rows),
		"drag to move | 1-9 states | +/- size | q",
	}
	if a.message != "" {
		lines = append(lines, trimToWidth(a.message, max(10, geometry.columns-1)))
	}
	for i, line := range lines {
		lines[i] = trimToWidth(line, geometry.columns-1)
	}

	var out strings.Builder
	out.WriteString(kitty.ClearAllImages())
	out.WriteString(kitty.EraseScreen())
	out.WriteString(kitty.CursorTo(1, 1))
	out.WriteString(strings.Join(lines, "\r\n"))
	out.WriteString(kitty.CursorTo(a.petRow, a.petCol))
	out.WriteString(kitty.TransmitFile(a.framePath(now), size.columns, size.rows))
	fmt.Print(out.String())
}

func (a *App) consumeInputBuffer() {
	for len(a.inputBuffer) > 0 {
		if match := sgrMouseRE.FindStringSubmatch(a.inputBuffer); len(match) == 5 {
			a.inputBuffer = a.inputBuffer[len(match[0]):]
			a.handleMouse(match)
			continue
		}

		switch {
		case strings.HasPrefix(a.inputBuffer, "\x1b[A"):
			a.inputBuffer = a.inputBuffer[3:]
		case strings.HasPrefix(a.inputBuffer, "\x1b[B"):
			a.inputBuffer = a.inputBuffer[3:]
		case strings.HasPrefix(a.inputBuffer, "\x1b[C"):
			a.inputBuffer = a.inputBuffer[3:]
		case strings.HasPrefix(a.inputBuffer, "\x1b[D"):
			a.inputBuffer = a.inputBuffer[3:]
		case strings.HasPrefix(a.inputBuffer, "\x1b") && len(a.inputBuffer) < 8:
			return
		default:
			char := a.inputBuffer[0]
			a.inputBuffer = a.inputBuffer[1:]
			a.handleKey(char)
		}
	}
}

func (a *App) handleKey(char byte) {
	keymap := map[byte]string{
		'1': "idle",
		'2': "move_left",
		'3': "move_right",
		'4': "wave",
		'5': "sit",
		'6': "sad",
		'7': "sleep",
		'8': "bounce",
		'9': "grumpy",
		' ': "sip",
	}

	switch char {
	case 'q', 3, 4:
		a.running = false
	case '+', '=':
		a.targetHeightPx = min(200, a.targetHeightPx+10)
		a.message = fmt.Sprintf("target height %dpx", a.targetHeightPx)
	case '-', '_':
		a.targetHeightPx = max(30, a.targetHeightPx-10)
		a.message = fmt.Sprintf("target height %dpx", a.targetHeightPx)
	default:
		if animation, ok := keymap[char]; ok {
			a.setAnimation(animation)
		}
	}
}

func (a *App) handleMouse(match []string) {
	button, _ := strconv.Atoi(match[1])
	x, _ := strconv.Atoi(match[2])
	y, _ := strconv.Atoi(match[3])
	final := match[4]

	released := final == "m" || button == 3
	pressed := button == 0 || button == 1 || button == 2
	draggingMotion := button >= 32 && button <= 34

	if released {
		a.pointer.down = false
		a.setAnimation("sip")
		a.message = fmt.Sprintf("dropped at x=%d y=%d", a.petCol, a.petRow)
		return
	}

	if pressed {
		a.pointer = pointer{x: x, y: y, down: true}
		a.setAnimation("wave")
		a.message = "drag to move"
		return
	}

	if draggingMotion && a.pointer.down {
		size := a.imageSize(terminalGeometryFor(int(os.Stdout.Fd())))
		previousX := a.pointer.x
		a.pointer = pointer{x: x, y: y, down: true}
		a.petCol = x - size.columns/2
		a.petRow = y - size.rows/2

		switch {
		case previousX != 0 && x < previousX:
			a.setAnimation("move_left")
		case previousX != 0 && x > previousX:
			a.setAnimation("move_right")
		default:
			a.setAnimation("wave")
		}
		a.message = fmt.Sprintf("dragging x=%d y=%d", a.petCol, a.petRow)
	}
}

func trimToWidth(value string, width int) string {
	if width < 1 || len(value) <= width {
		return value
	}
	return value[:width]
}
