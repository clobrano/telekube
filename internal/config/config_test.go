package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.KubectlBin != "kubectl" {
		t.Errorf("expected kubectl_bin to be 'kubectl', got '%s'", cfg.KubectlBin)
	}

	if cfg.Pager != "less" {
		t.Errorf("expected pager to be 'less', got '%s'", cfg.Pager)
	}

	if len(cfg.Tabs) != 3 {
		t.Errorf("expected 3 default tabs, got %d", len(cfg.Tabs))
	}

	if cfg.Tabs[0].Name != "Pods" {
		t.Errorf("expected default tab name to be 'Pods', got '%s'", cfg.Tabs[0].Name)
	}

	if cfg.Tabs[0].Command != "pods -A" {
		t.Errorf("expected default tab command to be 'pods -A', got '%s'", cfg.Tabs[0].Command)
	}

	// Check some keybindings
	if cfg.Keybindings.Quit != "q" {
		t.Errorf("expected quit keybinding to be 'q', got '%s'", cfg.Keybindings.Quit)
	}

	if cfg.Keybindings.Help != "?" {
		t.Errorf("expected help keybinding to be '?', got '%s'", cfg.Keybindings.Help)
	}
}

func TestLoadConfigWithDeleteTab(t *testing.T) {
	// Create a temporary config file with delete_tab keybinding
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
kubectl_bin: /usr/local/bin/kubectl
pager: bat
keybindings:
  delete_tab: "-"
tabs:
  - name: "My Pods"
    command: "get pods -n default"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Keybindings.DeleteTab != "-" {
		t.Errorf("expected delete_tab keybinding to be '-', got '%s'", cfg.Keybindings.DeleteTab)
	}
}

func TestLoadConfigDefaultDeleteTab(t *testing.T) {
	// Create a temporary config file WITHOUT delete_tab keybinding
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
kubectl_bin: /usr/local/bin/kubectl
pager: bat
tabs:
  - name: "My Pods"
    command: "get pods -n default"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Keybindings.DeleteTab != "-" {
		t.Errorf("expected default delete_tab keybinding to be '-', got '%s'", cfg.Keybindings.DeleteTab)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
kubectl_bin: /usr/local/bin/kubectl
pager: bat
tabs:
  - name: "My Pods"
    command: "get pods -n default"
  - name: "Deployments"
    command: "get deployments -A"
custom_commands:
  - name: "Copy Name"
    key: "y"
    command: "echo {{.name}}"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.KubectlBin != "/usr/local/bin/kubectl" {
		t.Errorf("expected kubectl_bin to be '/usr/local/bin/kubectl', got '%s'", cfg.KubectlBin)
	}

	if cfg.Pager != "bat" {
		t.Errorf("expected pager to be 'bat', got '%s'", cfg.Pager)
	}

	if len(cfg.Tabs) != 2 {
		t.Errorf("expected 2 tabs, got %d", len(cfg.Tabs))
	}

	if cfg.Tabs[0].Name != "My Pods" {
		t.Errorf("expected first tab name to be 'My Pods', got '%s'", cfg.Tabs[0].Name)
	}

	if cfg.Tabs[1].Command != "get deployments -A" {
		t.Errorf("expected second tab command to be 'get deployments -A', got '%s'", cfg.Tabs[1].Command)
	}

	if len(cfg.CustomCommands) != 1 {
		t.Errorf("expected 1 custom command, got %d", len(cfg.CustomCommands))
	}

	// Check that defaults are preserved for unspecified keybindings
	if cfg.Keybindings.Quit != "q" {
		t.Errorf("expected default quit keybinding to be preserved, got '%s'", cfg.Keybindings.Quit)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
tabs:
  - name: "Test"
    resource: [invalid yaml
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty kubectl_bin",
			config: &Config{
				KubectlBin: "",
				Tabs: []TabConfig{
					{Name: "Test", Command: "get pods"},
				},
			},
			wantErr: true,
		},
		{
			name: "no tabs",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs:       []TabConfig{},
			},
			wantErr: true,
		},
		{
			name: "tab without name",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs: []TabConfig{
					{Name: "", Command: "get pods"},
				},
			},
			wantErr: true,
		},
		{
			name: "tab without command",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs: []TabConfig{
					{Name: "Test", Command: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "custom command without name",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs: []TabConfig{
					{Name: "Test", Command: "get pods"},
				},
				CustomCommands: []CustomCommand{
					{Name: "", Key: "x", Command: "echo"},
				},
			},
			wantErr: true,
		},
		{
			name: "custom command without key",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs: []TabConfig{
					{Name: "Test", Command: "get pods"},
				},
				CustomCommands: []CustomCommand{
					{Name: "Test", Key: "", Command: "echo"},
				},
			},
			wantErr: true,
		},
		{
			name: "custom command without command",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs: []TabConfig{
					{Name: "Test", Command: "get pods"},
				},
				CustomCommands: []CustomCommand{
					{Name: "Test", Key: "x", Command: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "valid config with custom command",
			config: &Config{
				KubectlBin: "kubectl",
				Tabs: []TabConfig{
					{Name: "Test", Command: "get pods"},
				},
				CustomCommands: []CustomCommand{
					{Name: "Test", Key: "x", Command: "echo hello"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureConfigExists(t *testing.T) {
	// Save original home and restore after test
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	path, err := EnsureConfigExists()
	if err != nil {
		t.Fatalf("EnsureConfigExists failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".config", "telekube", "config.yaml")
	if path != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, path)
	}

	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load and validate the created config
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load created config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("created config is invalid: %v", err)
	}

	// Call again - should not error and return same path
	path2, err := EnsureConfigExists()
	if err != nil {
		t.Fatalf("second EnsureConfigExists failed: %v", err)
	}

	if path2 != path {
		t.Errorf("expected same path on second call, got '%s'", path2)
	}
}

func TestSaveConfigRemovesTab(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config with 3 tabs
	cfg := &Config{
		KubectlBin: "kubectl",
		Pager:      "less",
		Tabs: []TabConfig{
			{Name: "Pods", Command: "pods -A"},
			{Name: "Deployments", Command: "deployments -A"},
			{Name: "Services", Command: "services -A"},
		},
		Keybindings: DefaultConfig().Keybindings,
	}

	// Save it
	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load it back and verify
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(loaded.Tabs) != 3 {
		t.Errorf("expected 3 tabs, got %d", len(loaded.Tabs))
	}

	// Remove a tab and save again
	cfg.Tabs = cfg.Tabs[:len(cfg.Tabs)-1] // Remove last tab

	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("SaveConfig after removal failed: %v", err)
	}

	// Load and verify tab was removed
	loaded, err = LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig after removal failed: %v", err)
	}

	if len(loaded.Tabs) != 2 {
		t.Errorf("expected 2 tabs after removal, got %d", len(loaded.Tabs))
	}

	if len(loaded.Tabs) > 0 && loaded.Tabs[len(loaded.Tabs)-1].Name == "Services" {
		t.Error("Services tab should have been removed")
	}
}

func TestSaveConfigWritesValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		KubectlBin: "kubectl",
		Pager:      "less",
		Tabs: []TabConfig{
			{Name: "Pods", Command: "pods -A"},
		},
		Keybindings: DefaultConfig().Keybindings,
	}

	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if len(data) == 0 {
		t.Error("config file is empty")
	}

	// Verify YAML is valid by loading it
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("YAML is not valid: %v", err)
	}

	if loaded.KubectlBin != "kubectl" {
		t.Errorf("expected kubectl_bin to be 'kubectl', got '%s'", loaded.KubectlBin)
	}
}

func TestSaveConfigFileWriteError(t *testing.T) {
	// Use a non-existent directory that can't be created
	configPath := "/root/cannot/write/here/config.yaml"

	cfg := &Config{
		KubectlBin: "kubectl",
		Pager:      "less",
		Tabs: []TabConfig{
			{Name: "Pods", Command: "pods -A"},
		},
		Keybindings: DefaultConfig().Keybindings,
	}

	err := SaveConfig(cfg, configPath)
	if err == nil {
		t.Error("expected error for non-writable path, got nil")
	}
}
