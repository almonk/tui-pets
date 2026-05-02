package main

import (
	"flag"
	"fmt"
	"os"

	"tui-pets/internal/pets"
	"tui-pets/internal/tui"
)

func main() {
	extractOnly := flag.Bool("extract-only", false, "slice the spritesheet into the frame cache and exit")
	flag.Parse()

	petID := "boba"
	if flag.NArg() > 0 {
		petID = flag.Arg(0)
	}

	pet, err := pets.Load(petID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui-pets: %v\n", err)
		os.Exit(2)
	}

	app := tui.New(pet)
	if err := app.PrepareFrames(); err != nil {
		fmt.Fprintf(os.Stderr, "tui-pets: %v\n", err)
		os.Exit(2)
	}

	if *extractOnly {
		fmt.Printf("Extracted %d frames for %s.\n", len(app.Frames()), pet.DisplayName)
		return
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui-pets: %v\n", err)
		os.Exit(1)
	}
}
