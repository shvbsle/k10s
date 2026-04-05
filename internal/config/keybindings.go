package config

import (
	"maps"

	"k8s.io/utils/set"
)

const (
	ActionCommand           = "command"
	ActionEscape            = "escape"
	ActionFirstLine         = "first-line"
	ActionHelp              = "help"
	ActionLastLine          = "last-line"
	ActionNavigateBottom    = "navigate-bottom"
	ActionNavigateDown      = "navigate-down"
	ActionNavigateTop       = "navigate-top"
	ActionNavigateUp        = "navigate-up"
	ActionOpenPodLogs       = "open-pod-lods"
	ActionPageNext          = "next-page"
	ActionPagePrevious      = "previous-page"
	ActionQuit              = "quit"
	ActionResetView         = "reset-view"
	ActionResourceDescribe  = "resource-describe"
	ActionResourceEdit      = "resource-edit"
	ActionResourceYaml      = "resource-yaml"
	ActionScrollLeft        = "scroll-left"
	ActionScrollRight       = "scroll-right"
	ActionSearch            = "search"
	ActionSubmit            = "submit"
	ActionToggleAutoScroll  = "toggle-auto-scroll"
	ActionToggleFullsreen   = "toggle-fullscreen"
	ActionToggleLineNumbers = "toggle-line-numbers"
	ActionToggleTimestamps  = "toggle-timestamps"
	ActionToggleWordWrap    = "toggle-word-wrap"
)

type KeyBind map[string][]string

func defaultKeybinds() KeyBind {
	return KeyBind{
		ActionCommand:           []string{":"},
		ActionEscape:            []string{"esc", "escape"},
		ActionFirstLine:         []string{"g"},
		ActionHelp:              []string{"?"},
		ActionLastLine:          []string{"G"},
		ActionNavigateBottom:    []string{"J", "shift+down"},
		ActionNavigateDown:      []string{"j", "down"},
		ActionNavigateTop:       []string{"K", "shift+up"},
		ActionNavigateUp:        []string{"k", "up"},
		ActionOpenPodLogs:       []string{"L"},
		ActionPageNext:          []string{"l", "pgdown"},
		ActionPagePrevious:      []string{"h", "pgup"},
		ActionQuit:              []string{"q", "ctrl+c"},
		ActionResetView:         []string{"0"},
		ActionResourceDescribe:  []string{"d"},
		ActionResourceEdit:      []string{"e"},
		ActionResourceYaml:      []string{"y"},
		ActionScrollLeft:        []string{"left"},
		ActionScrollRight:       []string{"right"},
		ActionSearch:            []string{"/"},
		ActionSubmit:            []string{"enter"},
		ActionToggleAutoScroll:  []string{"s"},
		ActionToggleFullsreen:   []string{"f"},
		ActionToggleLineNumbers: []string{"n"},
		ActionToggleTimestamps:  []string{"t"},
		ActionToggleWordWrap:    []string{"w"},
	}
}

func (k KeyBind) For(action string, keys ...string) bool {
	if keySet, ok := k[action]; ok {
		return set.New(keySet...).HasAny(keys...)
	}
	return false
}

func (k KeyBind) setOverrides(other KeyBind) KeyBind {
	kb := defaultKeybinds()
	maps.Copy(kb, other)
	return kb
}

func (k KeyBind) validate() error {
	return nil
}
