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

	"os/exec"

	"regexp"

	"github.com/alexflint/go-arg"
)

var args struct {
	Programs []string `arg:"positional" help:"program's windows class name to start/focus. This is listed as part of 'wmctrl -lx'"`
}

func regexGroupedFind(expr *regexp.Regexp, str string) map[string]string {
	result := make(map[string]string)
	match := expr.FindStringSubmatch(str)
	if len(match) > 0 {
		for i, name := range expr.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}
	}
	return result
}

func getWindowIds(clsName string, allWindows string) map[string]string {
	matches := make(map[string]string)

	ptrn := `(?P<windowid>0x[\w]+)\s+(?P<desktop>[-\d]+)\s.*%s.*\s\s`
	for _, line := range strings.Split(strings.ReplaceAll(string(allWindows), "\r\n", "\n"), "\n") {
		regx := regexp.MustCompile(fmt.Sprintf(ptrn, clsName))
		groups := regexGroupedFind(regx, line)
		if len(groups) > 0 {
			matches[groups["desktop"]] = groups["windowid"]
		}
	}

	return matches
}

func focusWindow(windowClsName string) error {
	out, err := exec.Command("xdotool", "get_desktop").Output()
	if err != nil {
		return err
	}
	desktop := strings.TrimSpace(string(out))

	allWindows, err := exec.Command("wmctrl", "-lx").Output()
	if err != nil {
		return err
	}
	matches := getWindowIds(windowClsName, string(allWindows))

	if len(matches) < 0 {
		return errors.New("not found any window running")
	}
	if windowId, ok := matches[desktop]; ok {
		return exec.Command("wmctrl", "-i", "-a", windowId).Run()
	}

	if windowId, ok := matches["-1"]; ok {
		return exec.Command("wmctrl", "-i", "-a", windowId).Run()
	}

	return errors.New("not found window running")
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
		exec.Command(
			"nohup",
			"kioclient5",
			"exec",
			desktopEntry,
			// _out="/dev/null",
			// _err=current_log
		).Run()
	}
}
