# tui-pets

A renderer for Codex Pets spritesheets in your terminal.

## Run Boba

```bash
go run ./cmd/tui-pets boba
```

You can also point it at any installed pet folder or `pet.json` file:

```bash
go run ./cmd/tui-pets ~/.codex/pets/chefito/
go run ./cmd/tui-pets ~/.codex/pets/chefito/pet.json
```

The renderer uses the Kitty graphics protocol directly. Run it inside Kitty or a
Kitty-compatible terminal. ImageMagick's `magick` CLI is used once to slice the
spritesheet into cached PNG frames.

## Controls

- Drag with the mouse to move the pet around.
- `1`-`9` switches animation states.
- `space` plays a short emote.
- `+` and `-` to scale the pet.
- `q` exits.
