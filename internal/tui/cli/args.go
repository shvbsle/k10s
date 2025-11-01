package cli

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
		return ""
	}

	// Check for "-n <namespace>", "--namespace <namespace>", or "in <namespace>" patterns
	for i := range args {
		if (args[i] == "-n" || args[i] == "--namespace" || args[i] == "in") && i+1 < len(args) {
			ns := args[i+1]
			if ns == "all" {
				return ""
			}
			return ns
		}
	}

	// Check if first arg is "all"
	if args[0] == "all" {
		return ""
	}

	// Otherwise, treat first arg as namespace
	return args[0]
}
