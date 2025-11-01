package log

import "log"

func TUI(format string, args ...any) {
	log.Printf("TUI: "+format, args)
}
