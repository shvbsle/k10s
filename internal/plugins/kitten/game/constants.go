package game

import tl "github.com/JoelOtter/termloop"

const (
	GameTitle = "KITTEN CLIMBER"

	JumpVelocity = -0.5
	Gravity      = 0.0625
	MoveSpeed    = 5

	KittenWidth  = 5
	KittenHeight = 3

	PlatformHeight = 1

	FishWidth  = 3
	FishHeight = 1

	GameFPS = 120
)

var (
	KittenSprite = []string{
		"/\\_/\\",
		"(o.o)",
		" >^< ",
	}

	FishSprite = "><>"

	TitleScreen = []string{
		"╔═══════════════════════════════════════════════════════════════════════════╗",
		"║                                                                           ║",
		"║          ██╗  ██╗██╗████████╗████████╗███████╗███╗   ██╗███████╗          ║",
		"║          ██║ ██╔╝██║╚══██╔══╝╚══██╔══╝██╔════╝████╗  ██║██╔════╝          ║",
		"║          █████╔╝ ██║   ██║      ██║   █████╗  ██╔██╗ ██║███████╗          ║",
		"║          ██╔═██╗ ██║   ██║      ██║   ██╔══╝  ██║╚██╗██║╚════██║          ║",
		"║          ██║  ██╗██║   ██║      ██║   ███████╗██║ ╚████║███████║          ║",
		"║          ╚═╝  ╚═╝╚═╝   ╚═╝      ╚═╝   ╚══════╝╚═╝  ╚═══╝╚══════╝          ║",
		"║                                                                           ║",
		"║                    Infinite Runner - Collect Fish!                        ║",
		"║                                                                           ║",
		"║                         Arrow Keys: Move                                  ║",
		"║                         Space: Jump                                       ║",
		"║                         Ctrl+C: Back to k10s                              ║",
		"║                                                                           ║",
		"║                    Press SPACE to start...                                ║",
		"║                                                                           ║",
		"╚═══════════════════════════════════════════════════════════════════════════╝",
	}

	ColorKitten       = tl.ColorYellow
	ColorTitle        = tl.ColorYellow
	ColorPod          = tl.ColorCyan
	ColorNode         = tl.ColorGreen
	ColorService      = tl.ColorMagenta
	ColorControlPlane = tl.ColorRed
	ColorFish         = tl.ColorBlue
	ColorBackground   = tl.ColorBlack
	ColorText         = tl.ColorWhite
)

type GameState int

const (
	StateTitleScreen GameState = iota
	StatePlaying
	StateWin
	StateLose
)
