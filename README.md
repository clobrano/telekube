# telekube

A terminal UI for exploring and managing Kubernetes clusters. Browse resources, inspect details, and perform common kubectl operations through an interactive keyboard-driven interface.

## Features

- **Multi-tab resource browsing** - View Pods, Deployments, Services, and other resources in configurable tabs
- **Search tab** - Execute kubectl GET commands interactively from the first tab
- **Command-based tabs** - Configure tabs with any `kubectl get` command — filter by namespace, labels, status, or any other criteria
- **Runtime config editing** - Add new tabs or save commands to `config.yaml` without leaving the TUI
- **Fuzzy search** - Filter resources in real-time with intelligent matching
- **Multiple output formats** - View resources as table or YAML
- **Resource actions** - Describe, edit, delete resources; view logs
- **Context & namespace switching** - Quickly switch between clusters and namespaces
- **Multi-select** - Select multiple resources for bulk operations
- **Fully configurable** - Customize keybindings, tabs, and commands via YAML

## Installation

```bash
go install github.com/clobrano/telekube@latest
```

### From source

```bash
git clone https://github.com/clobrano/telekube.git
cd telekube
make install
```

### Requirements

- Go 1.21+
- kubectl configured with cluster access

## Usage

```bash
telekube
```

On first run, a default configuration file is created at `~/.config/telekube/config.yaml`.

## Visual Tour

### Initial Views

When you start Telekube, you'll see a list of resources organized in tabs. Here are the main views:

**Pods view** - Browse all pods across namespaces:
![Pods view](assets/01%20first%20start%20Pods%20view.png)

**Deployments view** - Monitor your deployments:
![Deployments view](assets/02%20first%20start%20Deployments%20view.png)

### Creating Custom Tabs

Add a new tab to monitor specific resources. First, enter the tab name:
![Create tab - name](assets/03%20Create%20new%20tab%20view%20-%20tab%20name.png)

Then enter your kubectl command:
![Create tab - command](assets/04%20Create%20new%20tab%20view%20-%20Command.png)

The new tab appears and starts populating with results:
![New tab empty](assets/05%20New%20Tab%20view%20-%20empty.png)
![New tab populating](assets/06%20New%20Tab%20view%20-%20populating.png)

> **💡 Tip:** Your new tabs are automatically saved to `~/.config/telekube/config.yaml` and will persist across sessions. See [Runtime Config Editing](#runtime-config-editing) for more details.

### Viewing Details

Access detailed information about your resources:

**Log view** - Stream and inspect pod logs:
![Logs part 1](assets/07%20Log%20view%201⁄2.png)
![Logs part 2](assets/08%20Log%20view%202⁄2.png)

**Description view** - Full resource details:
![Description view](assets/09%20Description%20view.png)

**Filter resources** - Use fuzzy search to find what you need:
![Filter view](assets/10%20Filter%20resource%20view.png)

## Search Tab

The first tab is always the **Search** tab, which lets you run kubectl GET commands interactively.

1. Navigate to the Search tab (press `1` or use `Tab`/`Shift+Tab`)
2. Press `Enter` to open the command input
3. Type a kubectl command (e.g., `get nodes -o wide`, `get pods -l app=nginx`)
4. Press `Enter` to execute and view results
5. Use `/` to filter the results with fuzzy search

The Search tab is useful for ad-hoc queries without needing to configure a new tab.

## Runtime Config Editing

You can add new tabs and save commands to `config.yaml` directly from the TUI — no editor required.

### Add a new tab (`+`)

Press `+` from any tab to open a two-step dialog:

1. Enter a name for the new tab (e.g. `Failing Pods`)
2. Enter a `kubectl get` command (e.g. `pods -A --field-selector=status.phase!=Running`)

The new tab is validated against the full config before `config.yaml` is written, so a bad command can never corrupt your configuration. On success the tab appears immediately and is focused.

### Save a tab command (`w`)

| Context | Behaviour |
|---------|-----------|
| **Config tab** (main view) | Saves the tab's current command to `config.yaml` immediately |
| **Config tab** (inside the `Enter` edit dialog) | Press `w` instead of `Enter` to apply the new command **and** save it in one step |
| **Search tab** | Opens a name prompt; saves the search command as a new permanent tab entry |

Changes are written atomically — the config is validated before the file is overwritten.

## Fuzzy Filter

Press `/` to activate fuzzy filter mode. The filter uses fuzzy matching to filter resources in real-time as you type.

### How fuzzy filter works

- **Case-insensitive** - `nginx` matches `NGINX`, `Nginx`, etc.
- **Non-contiguous matching** - Characters must appear in order but don't need to be adjacent. `ngx` matches `nginx`, `nxabc` matches `nginx-abc123`
- **Searches all columns** - Matches against name, status, namespace, or any visible column
- **Smart ranking** - Results are sorted by match quality:
  - Exact matches rank highest
  - Matches at the start of words rank higher
  - Consecutive character matches rank higher
  - Shorter matches rank higher than longer ones

### Filter controls

| Key | Action |
|-----|--------|
| `/` | Activate fuzzy filter |
| `Enter` | Confirm filter and return to list |
| `Esc` | Cancel filter and show all resources |

### Examples

| Query | Matches |
|-------|---------|
| `nginx` | `nginx-deployment`, `my-nginx-pod` |
| `ngx` | `nginx`, `nginx-abc123` |
| `run` | pods with status `Running` |
| `def` | resources in `default` namespace |

## Keyboard Shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `g` | Go to first item |
| `G` | Go to last item |
| `Tab` | Next tab |
| `Shift+Tab` | Previous tab |
| `1-9` | Jump to tab by number |

### Views

| Key | Action |
|-----|--------|
| `Enter` | View resource details (table format) |
| `Y` | View as YAML |
| `J` | View as JSON |
| `Esc` | Return to list view |

### Actions

| Key | Action |
|-----|--------|
| `d` | Describe resource |
| `l` | View logs (pods only) |
| `L` | Follow logs (pods only) |
| `D` | Delete resource (with confirmation) |
| `e` | Edit resource |
| `T` | Open system terminal |
| `R` | Rollout restart (deployments) |

### Selection

| Key | Action |
|-----|--------|
| `Space` | Toggle selection on current item |
| `a` | Select all |
| `A` | Deselect all |

### Config editing

| Key | Action |
|-----|--------|
| `+` | Add a new tab (prompts for name then command) |
| `Ctrl+W` | Save current tab command to `config.yaml` |
| `Ctrl+W` *(Search tab)* | Save search command as a new named tab |
| `Enter` *(edit dialog)* | Apply new command (run only) |
| `Ctrl+W` *(edit dialog)* | Apply new command **and** save to `config.yaml` |

### Other

| Key | Action |
|-----|--------|
| `c` | Switch kubectl context |
| `n` | Switch namespace |
| `/` | Search/filter |
| `r` | Refresh current view |
| `?` | Show help |
| `q` | Quit |

## Detail View

When viewing resource details (`Enter`, `Y`, or `J`), use these keys to navigate:

| Key | Action |
|-----|--------|
| `j` / `Down` | Scroll down |
| `k` / `Up` | Scroll up |
| `d` | Scroll half page down |
| `u` | Scroll half page up |
| `g` | Go to top |
| `G` | Go to bottom |
| `Esc` / `q` | Return to list |

## Configuration

Edit `~/.config/telekube/config.yaml` to customize:

```yaml
# Path to kubectl binary
kubectl_bin: "kubectl"

# Pager for long output (used by some actions)
pager: "less"

# Custom keybindings
keybindings:
  quit: "q"
  describe: "d"
  logs: "l"
  logs_follow: "L"
  delete: "D"
  edit: "e"
  terminal: "T"
  yaml_view: "Y"
  search: "/"
  refresh: "r"
  context: "c"
  namespace: "n"
  select: " "
  select_all: "a"
  deselect_all: "A"

# Configure resource tabs (command-based)
# Each tab runs a kubectl GET command (without the "kubectl" prefix, "get" is optional)
tabs:
  - name: "Pods"
    command: "pods -A"
  - name: "Running"
    command: "pods -A --field-selector=status.phase=Running"
  - name: "Deployments"
    command: "deployments -A"
  - name: "Services"
    command: "services -A"
  - name: "Nodes"
    command: "nodes -o wide"

# Tab options:
#   command: kubectl GET command (required)
#   sort_by: column name to sort by (e.g., "NAME", "AGE", "STATUS") or "creation_time" for the AGE column
#   sort_reverse: true to reverse the sort order (default: false)
#
# Examples of tab commands:
#   command: "pods -n kube-system"           # Specific namespace
#   command: "pods -l app=nginx"             # Filter by label
#   command: "events -A"                     # All events
#
# Examples with sorting:
#   - name: "Newest Pods"
#     command: "pods -A"
#     sort_by: "creation_time"               # Sort by AGE (newest first)
#   - name: "Deployments by Name"
#     command: "deployments -A"
#     sort_by: "NAME"
#     sort_reverse: true
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run with race detector
make build && ./telekube

# Install locally
make install
```

## License

MIT
