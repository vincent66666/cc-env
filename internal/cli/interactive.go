package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"cc-env/internal/profile"
)

type selectorAction int

const (
	selectorActionUp selectorAction = iota
	selectorActionDown
	selectorActionEnter
	selectorActionEdit
	selectorActionRename
	selectorActionRemove
	selectorActionQuit
)

const (
	clearScreenSequence      = "\x1b[H\x1b[2J"
	enterAlternateScreenMode = "\x1b[?1049h"
	exitAlternateScreenMode  = "\x1b[?1049l"
)

var (
	promptReader      io.Reader = os.Stdin
	promptWriter      io.Writer = os.Stdout
	promptInteractive           = func() bool {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return false
		}

		return stat.Mode()&os.ModeCharDevice != 0
	}
	startInteractiveSession = startInteractiveTerminalSession
	launchClaude            = runClaude
)

func selectorInteractive(stdout io.Writer) bool {
	if !promptInteractive() || !rawTerminalSupported() {
		return false
	}

	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return stat.Mode()&os.ModeCharDevice != 0
}

func runInteractiveStatus(paths Paths, selector statusSelector, stdout, stderr io.Writer) int {
	stdinFile, ok := promptReader.(*os.File)
	if !ok {
		_, _ = io.WriteString(stdout, selector.render())
		return 0
	}

	closeInteractive, err := startInteractiveSession(stdinFile, stdout)
	if err != nil {
		_, _ = io.WriteString(stdout, selector.render())
		return 0
	}
	defer closeInteractive()

	reader := bufio.NewReader(promptReader)
	for {
		_, _ = io.WriteString(stdout, clearScreenSequence)
		_, _ = io.WriteString(stdout, selector.render())

		action, err := readSelectorAction(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			_, _ = fmt.Fprintf(stderr, "读取交互输入失败：%v\n", err)
			return 1
		}

		switch action {
		case selectorActionUp:
			selector.moveUp()
		case selectorActionDown:
			selector.moveDown()
		case selectorActionEnter:
			closeInteractive()
			return switchProfile(paths, selector.selectedName(), nil, stderr)
		case selectorActionQuit:
			return 0
		}
	}
}

func runInteractiveList(paths Paths, menu listMenu, stdout, stderr io.Writer) int {
	stdinFile, ok := promptReader.(*os.File)
	if !ok {
		_, _ = io.WriteString(stdout, menu.render())
		return 0
	}

	closeInteractive, err := startInteractiveSession(stdinFile, stdout)
	if err != nil {
		_, _ = io.WriteString(stdout, menu.render())
		return 0
	}
	defer func() {
		if closeInteractive != nil {
			closeInteractive()
		}
	}()

	reader := bufio.NewReader(promptReader)
	for {
		_, _ = io.WriteString(stdout, clearScreenSequence)
		_, _ = io.WriteString(stdout, menu.render())

		action, err := readSelectorAction(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			_, _ = fmt.Fprintf(stderr, "读取交互输入失败：%v\n", err)
			return 1
		}

		switch action {
		case selectorActionUp:
			menu.moveUp()
		case selectorActionDown:
			menu.moveDown()
		case selectorActionEnter:
			if menu.mode == listMenuModeProfiles {
				if menuHasMissingCurrentProfile(menu) {
					closeInteractive()
					return switchProfile(paths, menu.selectedProfile(), nil, stderr)
				}
				menu.enterActions()
				continue
			}

			switch menu.mode {
			case listMenuModeActions:
				exitCode, done := executeListAction(paths, &menu, menu.selectedAction(), reader, stdinFile, stdout, stderr, &closeInteractive)
				if done {
					return exitCode
				}
			case listMenuModeDeleteConfirm:
				switch menu.selectedConfirmAction() {
				case listMenuConfirmDelete:
					exitCode, done := executeListDelete(paths, &menu, reader, stdinFile, stdout, stderr, &closeInteractive)
					if done {
						return exitCode
					}
				case listMenuConfirmCancel:
					menu.backToActions()
				}
			}
		case selectorActionEdit:
			if menu.mode == listMenuModeDeleteConfirm {
				continue
			}
			exitCode, done := executeListAction(paths, &menu, listMenuActionEdit, reader, stdinFile, stdout, stderr, &closeInteractive)
			if done {
				return exitCode
			}
		case selectorActionRename:
			if menu.mode == listMenuModeDeleteConfirm {
				continue
			}
			exitCode, done := executeListAction(paths, &menu, listMenuActionRename, reader, stdinFile, stdout, stderr, &closeInteractive)
			if done {
				return exitCode
			}
		case selectorActionRemove:
			if menu.mode == listMenuModeDeleteConfirm {
				continue
			}
			exitCode, done := executeListAction(paths, &menu, listMenuActionRemove, reader, stdinFile, stdout, stderr, &closeInteractive)
			if done {
				return exitCode
			}
		case selectorActionQuit:
			return 0
		}
	}
}

func executeListAction(
	paths Paths,
	menu *listMenu,
	action listMenuAction,
	reader *bufio.Reader,
	stdinFile *os.File,
	stdout, stderr io.Writer,
	closeInteractive *func(),
) (int, bool) {
	switch action {
	case listMenuActionSwitch:
		if selected := menu.selectedProfile(); selected != "" {
			if *closeInteractive != nil {
				(*closeInteractive)()
			}
			return switchProfile(paths, selected, nil, stderr), true
		}
	case listMenuActionEdit:
		selected := menu.selectedProfile()
		if selected == "" || profile.IsOfficialName(selected) {
			return 0, false
		}
		if *closeInteractive != nil {
			(*closeInteractive)()
		}

		exitCode := runEditWithPromptReader(paths, []string{selected}, reader, stdout, stderr)
		if exitCode != 0 {
			return exitCode, true
		}

		return resumeListSession(paths, menu, stdinFile, stdout, stderr, closeInteractive, selected, menu.index)
	case listMenuActionRename:
		selected := menu.selectedProfile()
		if selected == "" || profile.IsOfficialName(selected) {
			return 0, false
		}
		selectedIndex := menu.index

		if *closeInteractive != nil {
			(*closeInteractive)()
		}

		newName, err := promptRenameName(reader)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1, true
		}

		exitCode := runRename(paths, []string{selected, newName}, stdout, stderr)
		if exitCode != 0 {
			return exitCode, true
		}

		return resumeListSession(paths, menu, stdinFile, stdout, stderr, closeInteractive, newName, selectedIndex)
	case listMenuActionRemove:
		selected := menu.selectedProfile()
		if selected == "" || selected == menu.currentName || profile.IsOfficialName(selected) {
			return 0, false
		}
		menu.enterDeleteConfirm()
	case listMenuActionBack:
		menu.backToList()
	}

	return 0, false
}

func executeListDelete(
	paths Paths,
	menu *listMenu,
	reader *bufio.Reader,
	stdinFile *os.File,
	stdout, stderr io.Writer,
	closeInteractive *func(),
) (int, bool) {
	_ = reader
	selected := menu.selectedProfile()
	if selected == "" {
		return 0, false
	}
	selectedIndex := menu.index

	if *closeInteractive != nil {
		(*closeInteractive)()
	}

	exitCode := runRemove(paths, []string{selected}, stdout, stderr)
	if exitCode != 0 {
		return exitCode, true
	}

	return resumeListSession(paths, menu, stdinFile, stdout, stderr, closeInteractive, "", selectedIndex)
}

func resumeListSession(
	paths Paths,
	menu *listMenu,
	stdinFile *os.File,
	stdout, stderr io.Writer,
	closeInteractive *func(),
	selectedName string,
	selectedIndex int,
) (int, bool) {
	var err error
	*closeInteractive, err = startInteractiveSession(stdinFile, stdout)
	if err != nil {
		return 1, true
	}

	*menu, err = reloadListMenu(paths, selectedName, selectedIndex)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1, true
	}

	return 0, false
}

func startInteractiveTerminalSession(stdinFile *os.File, stdout io.Writer) (func(), error) {
	restore, err := makeRawTerminal(stdinFile)
	if err != nil {
		return nil, err
	}

	_, _ = io.WriteString(stdout, enterAlternateScreenMode)

	active := true
	return func() {
		if !active {
			return
		}
		active = false
		restore()
		_, _ = io.WriteString(stdout, exitAlternateScreenMode)
	}, nil
}

func reloadListMenu(paths Paths, selectedName string, selectedIndex int) (listMenu, error) {
	data, err := profile.LoadForList(paths.Profiles)
	if err != nil {
		return listMenu{}, err
	}

	menu := listMenu{
		profiles:     prioritizeCurrentProfile(profileNames(data.Profiles), data.Current),
		currentName:  data.Current,
		descriptions: profileDescriptions(data.Profiles),
	}

	if selectedName != "" {
		for i, name := range menu.profiles {
			if name == selectedName {
				menu.index = i
				return menu, nil
			}
		}
	}

	if selectedIndex >= len(menu.profiles) {
		selectedIndex = len(menu.profiles) - 1
	}
	if selectedIndex < 0 {
		selectedIndex = 0
	}
	menu.index = selectedIndex
	return menu, nil
}

func readSelectorAction(reader *bufio.Reader) (selectorAction, error) {
	for {
		key, err := reader.ReadByte()
		if err != nil {
			return selectorActionQuit, err
		}

		switch key {
		case 0x03:
			return selectorActionQuit, nil
		case 'e', 'E':
			return selectorActionEdit, nil
		case 'r', 'R':
			return selectorActionRename, nil
		case 'd', 'D':
			return selectorActionRemove, nil
		case 'q', 'Q':
			return selectorActionQuit, nil
		case '\r', '\n':
			return selectorActionEnter, nil
		case 0x1b:
			next, err := reader.ReadByte()
			if err != nil {
				return selectorActionQuit, err
			}
			if next != '[' {
				continue
			}

			arrow, err := reader.ReadByte()
			if err != nil {
				return selectorActionQuit, err
			}

			switch arrow {
			case 'A':
				return selectorActionUp, nil
			case 'B':
				return selectorActionDown, nil
			}
		}
	}
}

func menuHasMissingCurrentProfile(menu listMenu) bool {
	if menu.currentName == "" {
		return false
	}

	for _, name := range menu.profiles {
		if name == menu.currentName {
			return false
		}
	}

	return true
}
