package ui

type GUIOptions struct {
	AllowMultiple  bool
	ElevatedChild  bool
	NoAutoWorkflow bool
	Profile        string
}

func parseGUIOptions(args []string) GUIOptions {
	var opts GUIOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--allow-multiple":
			opts.AllowMultiple = true
		case "--elevated-child":
			opts.ElevatedChild = true
		case "--no-auto-workflow":
			opts.NoAutoWorkflow = true
		case "--profile":
			if i+1 < len(args) {
				opts.Profile = args[i+1]
				i++
			}
		}
	}
	return opts
}
