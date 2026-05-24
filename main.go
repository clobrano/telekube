package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clobrano/telekube/internal/config"
	"github.com/clobrano/telekube/internal/kubectl"
	"github.com/clobrano/telekube/internal/tui"
)

// Version information set by ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var customConfigPath string

	// Parse flags from os.Args
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--version", "-v":
			fmt.Printf("telekube %s (commit: %s, built: %s)\n", version, commit, date)
			os.Exit(0)
		case "--config", "-c":
			if i+1 >= len(os.Args) {
				fmt.Fprintf(os.Stderr, "Error: --config requires a path argument\n")
				os.Exit(1)
			}
			customConfigPath = os.Args[i+1]
			i++ // skip next arg
		case "--help", "-h":
			fmt.Println("Usage: telekube [flags]")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  -c, --config <path>   Path to config.yaml (default: ~/.config/telekube/config.yaml)")
			fmt.Println("  -v, --version          Show version information")
			fmt.Println("  -h, --help             Show this help message")
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\nRun 'telekube --help' for usage.\n", os.Args[i])
			os.Exit(1)
		}
	}

	var configPath string
	var err error

	if customConfigPath != "" {
		// Use the user-specified config path directly
		configPath = customConfigPath
	} else {
		// Ensure default config exists (create if needed)
		configPath, err = config.EnsureConfigExists()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error ensuring config exists: %v\n", err)
			os.Exit(1)
		}
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Create kubectl client
	k := kubectl.NewKubectl(cfg.KubectlBin)

	// Create and run the TUI
	model := tui.NewModel(cfg, k, configPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
