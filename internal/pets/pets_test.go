package pets

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMinimalPet(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pet.json"), []byte(`{
		"id": "chefito",
		"displayName": "Chefito",
		"description": "A tiny recipe-loving chef",
		"spritesheetPath": "spritesheet.webp"
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spritesheet.webp"), []byte("not-used-by-loader"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadPetDirectoryUsesInstalledPetDefaults(t *testing.T) {
	dir := writeMinimalPet(t)

	pet, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if pet.ID != "chefito" || pet.DisplayName != "Chefito" {
		t.Fatalf("unexpected pet identity: %#v", pet)
	}
	if pet.FrameWidth != 192 || pet.FrameHeight != 208 || pet.Columns != 8 || pet.Rows != 9 {
		t.Fatalf("default frame was not applied: %#v", pet)
	}
	if len(pet.Animations) == 0 || len(pet.Animations["idle"].Frames) == 0 {
		t.Fatalf("default animations were not applied: %#v", pet.Animations)
	}
}

func TestLoadPetJsonPathUsesContainingDirectory(t *testing.T) {
	dir := writeMinimalPet(t)

	pet, err := Load(filepath.Join(dir, "pet.json"))
	if err != nil {
		t.Fatal(err)
	}

	if pet.SpritesheetPath != filepath.Join(dir, "spritesheet.webp") {
		t.Fatalf("spritesheet path = %q, want %q", pet.SpritesheetPath, filepath.Join(dir, "spritesheet.webp"))
	}
}
