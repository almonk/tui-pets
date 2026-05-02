package kitty

import (
	"strings"
	"testing"
)

func TestMouseOnDoesNotEnableAllMotionTracking(t *testing.T) {
	command := MouseOn()
	if strings.Contains(command, "\x1b[?1003h") {
		t.Fatal("MouseOn enabled all-motion tracking")
	}
}
