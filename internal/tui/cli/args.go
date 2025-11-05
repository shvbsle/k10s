package cli

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// parseNamespace parses namespace from command arguments.
// Supports patterns:
//   - "-n <namespace>" or "--namespace <namespace>"
//   - "<namespace>" (direct)
//   - "in <namespace>"
//   - "all" or "-n all" (for all namespaces)
//
// Returns "" for all namespaces, or the specific namespace name.
func ParseNamespace(args []string) string {
	if len(args) == 0 {
		// No args means all namespaces
		return metav1.NamespaceAll
	}

	// Check for "-n <namespace>", "--namespace <namespace>", or "in <namespace>" patterns
	for i := range args {
		if (args[i] == "-n" || args[i] == "--namespace" || args[i] == "in") && i+1 < len(args) {
			ns := args[i+1]
			if ns == "all" {
				return metav1.NamespaceAll
			}
			return ns
		}
	}

	// Check if first arg is "all"
	if args[0] == "all" {
		return metav1.NamespaceAll
	}

	// Otherwise, treat first arg as namespace
	return args[0]
}

type Args struct {
	fields   []string
	trailing bool
}

// ParseArgs return the individual arguments in a command
func ParseArgs(command string) Args {
	return Args{
		fields:   strings.Fields(command),
		trailing: strings.HasSuffix(command, " " /* any whitespace? */),
	}
}

func (args Args) ReplaceLast(last string) Args {
	if len(args.fields) == 0 {
		return args
	}
	if args.trailing {
		// trailing means the user expects the autocomplete to be a completely
		// new argument to the command, instead of replacing the last string.
		args.fields = append(args.fields, last)
	} else {
		args.fields[len(args.fields)-1] = last
	}
	return args
}

func (args Args) AsList() []string {
	return args.fields
}
