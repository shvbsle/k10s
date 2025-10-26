package game

import tl "github.com/JoelOtter/termloop"

const (
	GameTitle = "KITTEN CLIMBER"

	JumpVelocity = -1.5
	Gravity      = 0.15
	MoveSpeed    = 1

	KittenWidth  = 5
	KittenHeight = 3

	PlatformHeight = 1

	FishWidth  = 3
	FishHeight = 1

	MinScreenWidth  = 60
	MinScreenHeight = 20
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
		"║                 Help the kittens reach the Control Plane!                 ║",
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
