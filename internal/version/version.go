package version

import (
	"fmt"
)

var Version = "source"

func String() string {
	return fmt.Sprintf("git-vibe version: %s\n", Version)
}
