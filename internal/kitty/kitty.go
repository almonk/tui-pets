package kitty

import (
	"encoding/base64"
	"fmt"
)

const (
	esc = "\x1b"
	st  = "\x1b\\"
)

func ClearAllImages() string {
	return esc + "_Ga=d,d=A,q=2;" + st
}

func TransmitFile(path string, columns, rows int) string {
	payload := base64.StdEncoding.EncodeToString([]byte(path))
	return fmt.Sprintf("%s_Ga=T,t=f,f=100,c=%d,r=%d,q=2;%s%s", esc, columns, rows, payload, st)
}

func CursorTo(row, column int) string {
	if row < 1 {
		row = 1
	}
	if column < 1 {
		column = 1
	}
	return fmt.Sprintf("%s[%d;%dH", esc, row, column)
}

func AltScreenOn() string {
	return esc + "[?1049h" + esc + "[?25l"
}

func AltScreenOff() string {
	return ClearAllImages() + esc + "[?25h" + esc + "[?1049l"
}

func MouseOn() string {
	return esc + "[?1000h" + esc + "[?1002h" + esc + "[?1006h"
}

func MouseOff() string {
	return esc + "[?1006l" + esc + "[?1003l" + esc + "[?1002l" + esc + "[?1000l"
}

func EraseScreen() string {
	return esc + "[2J"
}
