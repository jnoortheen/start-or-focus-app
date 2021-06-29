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
	"os/user"

	"regexp"

	"github.com/alexflint/go-arg"
)

var args struct {
	Programs   []string `arg:"positional" help:"program's windows class name to start/focus. This is listed as part of 'wmctrl -lx'"`
	AnyDesktop bool     `help:"activate window in any desktop."`
	Exec       bool     `help:"Launch by calling executable directly instead of using .desktop file."`
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

	windowId := ""
	if wid, ok := matches["-1"]; ok {
		windowId = wid
	}
	if args.AnyDesktop && len(matches) > 0 {
		for _, wid := range matches {
			windowId = wid
			break
		}
	}

	if len(windowId) > 0 {
		return exec.Command("wmctrl", "-i", "-a", windowId).Run()
	}

	return errors.New("not found window running")
}

func searchDesktopEntry(program string) (string, error) {
	dirsEnv := os.Getenv("XDG_DATA_DIRS")
	dirs := filepath.SplitList(dirsEnv)

	usr, _ := user.Current()
	homeDir := filepath.Join(usr.HomeDir, ".local", "share")
	dirs = append(dirs, homeDir)

	pattern := "*" + program + "*.desktop"
	for _, dir := range dirs {
		app_dir := filepath.Join(dir, "applications", pattern)
		fmt.Printf("Searching in %s \n", app_dir)
		matches, _ := filepath.Glob(app_dir)
		for _, path := range matches {
			if !strings.Contains(path, "-handler") {
				return path, nil
			}
		}
	}
	return "", errors.New("no file found")
}

func launchByKioClient(program string) {
	desktopEntry, err := searchDesktopEntry(program)
	if err == nil {
		fmt.Println("Found program entry " + desktopEntry)
		err := exec.Command(
			"nohup",
			"kioclient5",
			"exec",
			desktopEntry,
		).Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func launchByDbus(program string) {
	out, err := exec.Command(
		"qdbus", "org.kde.klauncher5",
		"/KLauncher",
		"exec_blind",
		program,
	).Output()
	if err != nil {
		fmt.Println("Error during klaunch: ", string(out), err)
	}
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
		} else {
			fmt.Println(err)
		}
	}

	program := args.Programs[0]
	fmt.Println("Starting the program " + program)
	if args.Exec {
		err := exec.Command(
			program,
		).Run()
		if err != nil {
			fmt.Println("Error during app launch: ", err)
		}
	} else {
		launchByKioClient(program)
	}
}
