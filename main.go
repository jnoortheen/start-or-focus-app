///usr/bin/env go run "$0" "$@" ; exit "$?"

package main

// https://blog.cloudflare.com/using-go-as-a-scripting-language-in-linux/
// using gorun to run the files correctly
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/magefile/mage/sh"
)

var args struct {
	Programs []string `arg:"positional" help:"name of program to start/focus"`
}

func focusWindow(program string) error {
	return sh.Run("wmctrl", "-x", "-a", program)
}

func searchDesktopEntry(program string) (string, error) {
	dirsEnv := os.Getenv("XDG_DATA_DIRS")
	dirs := filepath.SplitList(dirsEnv)
	pattern := "*" + program + "*.desktop"
	for _, dir := range dirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "applications", pattern))
		for _, path := range matches {
			if !strings.Contains(path, "-handler") {
				return path, nil
			}
		}
	}
	return "", errors.New("no file found")
}

func main() {
	arg.MustParse(&args)

	if len(args.Programs) == 0 {
		fmt.Println("required program names one or more")
		return
	}

	// try to focus the given program
	for _, prog := range args.Programs {
		err := focusWindow(prog)
		if err == nil {
			return
		}
	}

	program := args.Programs[0]
	fmt.Println("Starting the program " + program)

	desktopEntry, derr := searchDesktopEntry(program)
	if derr == nil {
		sh.Run(
			"nohup",
			"kioclient5",
			"exec",
			desktopEntry,
			// _out="/dev/null",
			// _err=current_log
		)
	}
}
