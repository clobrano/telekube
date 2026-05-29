package config

import "time"

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() *Config {
	return &Config{
		KubectlBin:           "kubectl",
		Pager:                "less",
		RefreshInterval:      5 * time.Second,
		LogsFollowInNewShell: true,
		Keybindings: Keybindings{
			Quit:            "q",
			Help:            "?",
			Refresh:         "r",
			Search:          "/",
			Describe:        "d",
			Logs:   "L",
			Delete: "D",
			Edit:            "e",
			Terminal:        "T",
			PortForward:     "p",
			Scale:           "s",
			RolloutRestart:  "R",
			YAMLView:        "Y",
			JSONView:        "J",
			SwitchNamespace: "n",
			SwitchContext:   "C",
			CopyData:        "c",
			MultiSelect:     " ",
			Enter:           "enter",
			Escape:          "esc",
			Up:              "up",
			Down:            "down",
			TabNext:         "tab",
			TabPrev:         "shift+tab",
			DeleteTab:       "-",
		},
		Tabs: []TabConfig{
			{
				Name:    "Pods",
				Command: "pods -A",
			},
			{
				Name:    "Deployments",
				Command: "deployments -A",
			},
			{
				Name:    "Services",
				Command: "services -A",
			},
		},
		CustomCommands: []CustomCommand{},
	}
}
