package tui

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clobrano/telekube/internal/actions"
	"github.com/clobrano/telekube/internal/config"
	"github.com/clobrano/telekube/internal/kubectl"
	"github.com/clobrano/telekube/internal/tui/components/detail"
	"github.com/clobrano/telekube/internal/tui/components/dialog"
	"github.com/clobrano/telekube/internal/tui/components/header"
	"github.com/clobrano/telekube/internal/tui/components/list"
	"github.com/clobrano/telekube/internal/tui/components/search"
	"github.com/clobrano/telekube/internal/tui/components/tabs"
)

// ViewState represents the current view mode of the application
type ViewState int

const (
	ViewList ViewState = iota
	ViewDetail
	ViewSearch
	ViewConfirm
	ViewSelector
	ViewInput
	ViewHelp
	ViewSearchTab // Search tab command input mode
)

// SearchTabIndex is the index of the special Search tab (always first)
const SearchTabIndex = 0

// normalizeGetCommand strips a leading "get" verb from a command if present,
// so that both "pods -A" and "get pods -A" (legacy configs) are handled uniformly.
func normalizeGetCommand(command string) string {
	trimmed := strings.TrimSpace(command)
	if strings.HasPrefix(trimmed, "get ") {
		return strings.TrimSpace(trimmed[4:])
	}
	return trimmed
}

// wrapAtWidth word-wraps text at the given column width, preserving words.
// Returns the text unchanged when width is 0 or the text already fits.
func wrapAtWidth(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}
	words := strings.Fields(text)
	var sb strings.Builder
	lineLen := 0
	for i, word := range words {
		if i == 0 {
			sb.WriteString(word)
			lineLen = len(word)
		} else if lineLen+1+len(word) <= width {
			sb.WriteByte(' ')
			sb.WriteString(word)
			lineLen += 1 + len(word)
		} else {
			sb.WriteByte('\n')
			sb.WriteString(word)
			lineLen = len(word)
		}
	}
	return sb.String()
}

// getNamespaceFromCommand extracts the explicit namespace from a kubectl command string.
// Returns "" if -A/--all-namespaces is used or no namespace flag is present.
func getNamespaceFromCommand(command string) string {
	parts := strings.Fields(command)
	for i, part := range parts {
		if part == "-A" || part == "--all-namespaces" {
			return ""
		}
		if (part == "-n" || part == "--namespace") && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.HasPrefix(part, "--namespace=") {
			return strings.TrimPrefix(part, "--namespace=")
		}
	}
	return ""
}

// getResourceTypeFromCommand extracts the resource type from a kubectl command string.
// Handles both "pods -A" and legacy "get pods -A" formats.
func getResourceTypeFromCommand(command string) string {
	normalized := normalizeGetCommand(command)
	parts := strings.Fields(normalized)
	// The first non-flag token is the resource type
	for _, part := range parts {
		if !strings.HasPrefix(part, "-") {
			return part
		}
	}
	return ""
}

// getCurrentTabCommand returns the command for the current tab
func (m *Model) getCurrentTabCommand() string {
	if m.currentTab == SearchTabIndex {
		return m.searchCommand
	}
	configTabIndex := m.currentTab - 1
	if configTabIndex < 0 || configTabIndex >= len(m.config.Tabs) {
		return ""
	}
	return m.config.Tabs[configTabIndex].Command
}

// getCurrentResourceType returns the resource type for the current tab
func (m *Model) getCurrentResourceType() string {
	return getResourceTypeFromCommand(m.getCurrentTabCommand())
}

// TabSortState holds runtime sort configuration for a single tab.
type TabSortState struct {
	SortBy      string
	SortReverse bool
}

// Model is the main application model for the TUI
type Model struct {
	config     *config.Config
	kubectl    *kubectl.Kubectl
	configPath string

	// UI state
	viewState    ViewState
	currentTab   int
	width        int
	height       int

	// Components
	header   *header.Model
	tabs     *tabs.Model
	list     *list.Model
	search   *search.Model
	detail   *detail.Model
	confirm  *dialog.ConfirmModel
	selector *dialog.SelectorModel
	input    *dialog.InputModel

	// Pending action state
	pendingAction  string
	pendingNames   []string
	pendingNs      string
	pendingTabName string // name collected during multi-step add-tab flow

	// Data state
	currentContext   string
	currentNamespace string
	resources        *kubectl.TableData
	selectedIndex    int

	// Search tab state
	searchInput   *dialog.InputModel
	searchCommand string // Last executed search command

	// Per-tab search state: saves the last confirmed filter for each tab index
	tabSearchStates map[int]string

	// Per-tab runtime sort state (set via the 's' shortcut; overrides config sort)
	tabRuntimeSorts map[int]TabSortState
	// Maps sort option label → TabSortState for the currently open sort selector
	sortOptionMap map[string]TabSortState

	// Loading state
	loading bool
	spinner spinner.Model

	// Error state
	lastError error

	// Refresh timing
	lastRefreshTime time.Time
}

// NewModel creates a new application model
func NewModel(cfg *config.Config, k *kubectl.Kubectl, configPath string) *Model {
	// Extract tab names from config, prepending Search tab
	tabNames := make([]string, len(cfg.Tabs)+1)
	tabNames[0] = "Search"
	for i, tab := range cfg.Tabs {
		tabNames[i+1] = tab.Name
	}

	// Create spinner for loading indicator
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return &Model{
		config:          cfg,
		kubectl:         k,
		configPath:      configPath,
		viewState:       ViewList,
		currentTab:      1, // Start on first configured tab, not Search
		header:          header.New("", "", 0),
		tabs:            tabs.New(tabNames, 1),
		list:            list.New([]string{}, [][]string{}),
		search:          search.New(),
		detail:          detail.New("", "", detail.FormatTable),
		searchInput:     dialog.NewInput("kubectl get", "pods -A"),
		tabSearchStates: make(map[int]string),
		tabRuntimeSorts: make(map[int]TabSortState),
		loading:         true,
		spinner:         s,
	}
}

// tickCmd returns a command that sends a TickMsg after the configured refresh interval.
func (m *Model) tickCmd() tea.Cmd {
	if m.config.RefreshInterval <= 0 {
		return nil
	}
	return tea.Tick(m.config.RefreshInterval, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}

// subTickCmd returns a command that sends a SubTickMsg every second to update the countdown display.
func (m *Model) subTickCmd() tea.Cmd {
	if m.config.RefreshInterval <= 0 {
		return nil
	}
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return SubTickMsg{}
	})
}

// Init returns the initial command to run when the program starts
func (m *Model) Init() tea.Cmd {
	m.lastRefreshTime = time.Now()
	m.header.SetRefreshInterval(m.config.RefreshInterval)
	m.header.SetRefreshRemaining(m.config.RefreshInterval)
	return tea.Batch(
		m.loadContext(),
		m.loadResources(),
		m.tickCmd(),
		m.subTickCmd(),
		m.spinner.Tick,
	)
}

// startLoading sets loading state and returns the cmd wrapped with spinner tick
func (m *Model) startLoading(cmd tea.Cmd) tea.Cmd {
	m.loading = true
	return tea.Batch(cmd, m.spinner.Tick)
}

// Update handles all messages and updates the model state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Forward spinner tick messages
	if _, ok := msg.(spinner.TickMsg); ok {
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Handle countdown sub-tick: update remaining time display every second.
	if _, ok := msg.(SubTickMsg); ok {
		remaining := m.config.RefreshInterval - time.Since(m.lastRefreshTime)
		m.header.SetRefreshRemaining(remaining)
		return m, m.subTickCmd()
	}

	// Handle auto-refresh tick: always reschedule, but only refresh in list view
	// when the user is not actively typing a filter.
	if _, ok := msg.(TickMsg); ok {
		m.lastRefreshTime = time.Now()
		m.header.SetRefreshRemaining(m.config.RefreshInterval)
		next := m.tickCmd()
		if m.viewState == ViewList && !m.search.IsActive() {
			return m, tea.Batch(m.loadResources(), next)
		}
		return m, next
	}

	// Handle confirm dialog state
	if m.viewState == ViewConfirm && m.confirm != nil {
		m.confirm, _ = m.confirm.Update(msg)
		switch m.confirm.Result() {
		case dialog.ConfirmYes:
			m.viewState = ViewList
			return m, m.startLoading(m.executeDelete())
		case dialog.ConfirmNo:
			m.viewState = ViewList
			m.pendingAction = ""
			m.pendingNames = nil
			m.pendingNs = ""
		}
		return m, nil
	}

	// Handle selector dialog state
	if m.viewState == ViewSelector && m.selector != nil {
		m.selector, _ = m.selector.Update(msg)
		switch m.selector.Result() {
		case dialog.SelectorSelected:
			m.viewState = ViewList
			return m, m.handleSelectorResult()
		case dialog.SelectorCancelled:
			m.viewState = ViewList
			m.pendingAction = ""
		}
		return m, nil
	}

	// Handle detail view state
	if m.viewState == ViewDetail {
		// When the detail search input is active, forward messages to it
		if m.detail.IsSearching() {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				m.detail.ClearSearch()
				m.viewState = ViewList
				return m, nil
			case "esc":
				// If a search query is active, clear it first; otherwise go back
				if m.detail.HasSearchQuery() {
					m.detail.ClearSearch()
				} else {
					m.viewState = ViewList
				}
				return m, nil
			case "/":
				m.detail.StartSearch()
				return m, nil
			case "n":
				m.detail.NextMatch()
			case "N":
				m.detail.PrevMatch()
			case "j", "down":
				m.detail.ScrollDown()
			case "k", "up":
				m.detail.ScrollUp()
			case "g":
				m.detail.ScrollToTop()
			case "G":
				m.detail.ScrollToBottom()
			case "ctrl+d", "pgdown":
				m.detail.PageDown()
			case "ctrl+u", "pgup":
				m.detail.PageUp()
			case "Y":
				return m, m.startLoading(m.loadResourceDetail(detail.FormatYAML))
			case "J":
				return m, m.startLoading(m.loadResourceDetail(detail.FormatJSON))
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.detail.SetSize(msg.Width, msg.Height-2)
		case DetailLoadedMsg:
			m.loading = false
			m.detail.SetContent(msg.ResourceName, msg.Content, detail.Format(msg.Format))
		}
		return m, nil
	}

	// Handle Search tab input mode
	if m.viewState == ViewSearchTab {
		// Intercept "w" when editing an existing config tab command (no actionID):
		// apply the typed command AND immediately save to config.yaml.
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "w" &&
			m.searchInput.ActionID() == "" && m.currentTab != SearchTabIndex {
			newValue := m.searchInput.Value()
			m.searchInput.Reset()
			m.viewState = ViewList
			configTabIndex := m.currentTab - 1
			if configTabIndex >= 0 && configTabIndex < len(m.config.Tabs) {
				m.config.Tabs[configTabIndex].Command = normalizeGetCommand(newValue)
			}
			return m, tea.Batch(m.startLoading(m.loadResources()), m.saveCurrentTabToConfig())
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)

		switch m.searchInput.Result() {
		case dialog.InputSubmitted:
			newValue := m.searchInput.Value()
			actionID := m.searchInput.ActionID()
			m.searchInput.Reset()
			m.viewState = ViewList

			switch actionID {
			case "add-tab-name":
				m.pendingTabName = strings.TrimSpace(newValue)
				if m.pendingTabName == "" {
					return m, nil
				}
				inp := dialog.NewInput("New Tab: Command", "pods -A")
				inp.WithHint(fmt.Sprintf("kubectl get command for tab %q  (e.g. pods -A, deployments -n default)", m.pendingTabName))
				inp.WithActionID("add-tab-command")
				inp.SetSize(m.width, m.height)
				m.searchInput = inp
				m.viewState = ViewSearchTab
				return m, nil

			case "add-tab-command":
				return m, m.addNewTab(m.pendingTabName, newValue)

			case "save-search-name":
				return m, m.saveSearchAsTab(newValue)

			default:
				// Original enter-key behavior: execute or update command
				if m.currentTab == SearchTabIndex {
					m.searchCommand = newValue
					return m, m.startLoading(m.executeSearchCommand())
				}
				configTabIndex := m.currentTab - 1
				if configTabIndex >= 0 && configTabIndex < len(m.config.Tabs) {
					m.config.Tabs[configTabIndex].Command = newValue
				}
				return m, m.startLoading(m.loadResources())
			}

		case dialog.InputCancelled:
			m.viewState = ViewList
			m.searchInput.Reset()
		}

		// Handle window resize in search input mode
		if wmsg, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = wmsg.Width
			m.height = wmsg.Height
			m.searchInput.SetSize(wmsg.Width, wmsg.Height)
		}

		return m, cmd
	}

	// If search is active (typing mode), forward messages to search component
	if m.search.IsActive() {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)

		// Update list with filtered results
		if m.resources != nil {
			filtered := m.search.FilteredItems()
			m.list.SetItems(m.resources.Headers, filtered)
		}

		// Check if search was fully cleared (Esc pressed or Enter on empty input)
		if !m.search.IsActive() && !m.search.IsFiltered() && m.resources != nil {
			delete(m.tabSearchStates, m.currentTab)
			m.list.SetItems(m.resources.Headers, m.resources.Rows)
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Clear an active filter (confirmed via Enter)
			if m.search.IsFiltered() {
				m.search.Deactivate()
				delete(m.tabSearchStates, m.currentTab)
				if m.resources != nil {
					m.list.SetItems(m.resources.Headers, m.resources.Rows)
				}
			}
			return m, nil
		case "/":
			// Activate search
			if m.resources != nil {
				m.search.SetItems(m.resources.Rows)
			}
			m.search.Activate()
			return m, nil
		case "enter":
			// Open command input dialog pre-filled with current command
			var currentCmd, hint string
			if m.currentTab == SearchTabIndex {
				currentCmd = m.searchCommand
				hint = "e.g. pods -A, deployments -n default, services -l app=web"
			} else {
				configTabIndex := m.currentTab - 1
				if configTabIndex >= 0 && configTabIndex < len(m.config.Tabs) {
					currentCmd = m.config.Tabs[configTabIndex].Command
				}
				hint = "e.g. pods -A, deployments -n default"
			}
			inp := dialog.NewInput("kubectl get", "pods -A")
			inp.WithHint(hint)
			inp.SetValue(currentCmd)
			if m.currentTab != SearchTabIndex {
				inp.WithExtraHints("[w] apply + save to config.yaml")
			}
			inp.SetSize(m.width, m.height)
			m.searchInput = inp
			m.viewState = ViewSearchTab
			return m, nil

		case "+":
			// Open dialog to add a new tab to config
			inp := dialog.NewInput("New Tab: Name", "e.g. My Pods")
			inp.WithHint("Enter a name for the new tab, then press Enter")
			inp.WithActionID("add-tab-name")
			inp.SetSize(m.width, m.height)
			m.searchInput = inp
			m.viewState = ViewSearchTab
			return m, nil

		case "w":
			// Save current tab's command to config.yaml
			if m.currentTab == SearchTabIndex {
				if m.searchCommand == "" {
					return m, nil
				}
				inp := dialog.NewInput("Save Search as Tab", "e.g. My Pods")
				inp.WithHint(fmt.Sprintf("Name for: kubectl get %s", normalizeGetCommand(m.searchCommand)))
				inp.WithActionID("save-search-name")
				inp.SetSize(m.width, m.height)
				m.searchInput = inp
				m.viewState = ViewSearchTab
			} else {
				return m, m.saveCurrentTabToConfig()
			}
			return m, nil
		case "Y":
			// Show detail view (YAML format)
			return m, m.startLoading(m.loadResourceDetail(detail.FormatYAML))
		case "J":
			// Show detail view (JSON format)
			return m, m.startLoading(m.loadResourceDetail(detail.FormatJSON))
		case "tab", "right", "l":
			m.saveCurrentTabSearch()
			m.tabs.Next()
			m.currentTab = m.tabs.Active()
			return m, m.handleTabChange()
		case "shift+tab", "left", "h":
			m.saveCurrentTabSearch()
			m.tabs.Previous()
			m.currentTab = m.tabs.Active()
			return m, m.handleTabChange()
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx < m.tabs.Count() {
				m.saveCurrentTabSearch()
				m.tabs.SetActive(idx)
				m.currentTab = idx
				return m, m.handleTabChange()
			}
		case "j", "down":
			m.list.MoveDown()
		case "k", "up":
			m.list.MoveUp()
		case "g":
			m.list.MoveToTop()
		case "G":
			m.list.MoveToBottom()
		case "r":
			return m, m.startLoading(m.loadResources())
		case " ":
			// Toggle selection of current item
			m.list.ToggleSelect()
		case "a":
			// Select all items
			m.list.SelectAll()
		case "A":
			// Deselect all items
			m.list.DeselectAll()
		case "s":
			// Open sort selector
			m.openSortSelector()
			return m, nil
		case "d":
			// Describe selected resource(s)
			return m, m.startLoading(m.describeResources())
		case "L":
			// View logs (for pods) - show selector with log options
			return m, m.startLoading(m.viewLogs())
		case "D":
			// Delete selected resource(s) - show confirmation
			return m, m.confirmDelete()
		case "e":
			// Edit resource
			return m, m.editResource()
		case "x":
			// Exec into pod
			return m, m.execIntoPod()
		case "R":
			// Rollout restart
			return m, m.startLoading(m.rolloutRestart())
		case "c":
			// Switch context
			return m, m.startLoading(m.showContextSelector())
		case "n":
			// Switch namespace
			return m, m.startLoading(m.showNamespaceSelector())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.tabs.SetWidth(msg.Width)
		m.search.SetWidth(msg.Width)
		// List height = total height - header (1) - tabs (1) - search (1 if active or filtered) - footer area (4: \n\n + up to 2 footer lines)
		listHeight := msg.Height - 6
		if m.search.IsActive() || m.search.IsFiltered() {
			listHeight--
		}
		if listHeight < 3 {
			listHeight = 3
		}
		m.list.SetSize(msg.Width, listHeight)

	case ContextLoadedMsg:
		m.currentContext = msg.Context
		m.currentNamespace = msg.Namespace
		m.header.SetContext(msg.Context)
		m.header.SetNamespace(msg.Namespace)
		m.loading = false

	case ResourcesLoadedMsg:
		m.loading = false
		m.resources = msg.Data
		m.lastError = nil
		if msg.Data != nil {
			m.applyTabSort(msg.Data)
			if m.search.IsFiltered() {
				// Refresh on same tab: re-apply the active filter to the new data
				m.search.SetItems(msg.Data.Rows)
				m.list.UpdateItems(msg.Data.Headers, m.search.FilteredItems())
			} else if savedQuery := m.tabSearchStates[m.currentTab]; savedQuery != "" {
				// Returning to a tab that had a saved filter: restore it
				m.search.SetItems(msg.Data.Rows)
				m.search.RestoreFilter(savedQuery)
				m.list.UpdateItems(msg.Data.Headers, m.search.FilteredItems())
			} else {
				m.list.UpdateItems(msg.Data.Headers, msg.Data.Rows)
			}
		}

	case ErrorMsg:
		m.loading = false
		m.lastError = msg.Err

	case DescribeLoadedMsg:
		m.loading = false
		m.viewState = ViewDetail
		title := "Describe"
		if len(msg.ResourceNames) == 1 {
			title = msg.ResourceNames[0]
		} else if len(msg.ResourceNames) > 1 {
			title = "Multiple Resources"
		}
		m.detail.SetContent(title, msg.Content, detail.FormatTable)
		m.detail.SetSize(m.width, m.height-2)

	case LogsLoadedMsg:
		m.loading = false
		m.viewState = ViewDetail
		title := msg.PodName
		if msg.Container != "" {
			title = msg.PodName + "/" + msg.Container
		}
		m.detail.SetContent(title+" (logs)", msg.Content, detail.FormatTable)
		m.detail.SetSize(m.width, m.height-2)

	case LogsFollowMsg:
		if msg.NewShell {
			// Launch follow logs in a new terminal shell (non-blocking)
			cmd := makeKubectlLogsFollowCmd(m.kubectl.BinaryPath(), msg.PodName, msg.Namespace, msg.Container)
			launchInNewShell(cmd.Args)
			return m, nil
		}
		return m, tea.ExecProcess(
			makeKubectlLogsFollowCmd(m.kubectl.BinaryPath(), msg.PodName, msg.Namespace, msg.Container),
			func(err error) tea.Msg {
				return RefreshMsg{}
			},
		)

	case ContainersLoadedMsg:
		m.loading = false
		m.pendingNames = []string{msg.PodName}
		m.pendingNs = msg.Namespace
		m.pendingAction = "log-options"

		// Build log option entries: for each container, add a plain log
		// and a --follow variant, sorted alphabetically.
		var items []string
		for _, c := range msg.Containers {
			items = append(items, fmt.Sprintf("log %s", c))
			items = append(items, fmt.Sprintf("log %s --follow", c))
		}
		sort.Strings(items)

		m.selector = dialog.NewSelector("Select", items)
		m.selector.SetSize(m.width, m.height)
		m.viewState = ViewSelector

	case RefreshMsg:
		return m, tea.Batch(m.loadContext(), m.loadResources())

	case TabAddedMsg:
		m.config.Tabs = append(m.config.Tabs, config.TabConfig{
			Name:    msg.Name,
			Command: msg.Command,
		})
		m.tabs.AddTab(msg.Name)
		newTabIndex := len(m.config.Tabs) // Search is 0, config tabs start at 1
		m.saveCurrentTabSearch()
		m.tabs.SetActive(newTabIndex)
		m.currentTab = newTabIndex
		return m, m.startLoading(m.loadResources())

	case ConfigSavedMsg:
		// silent success
		return m, nil
	}

	return m, nil
}

// View renders the current state
func (m *Model) View() string {
	// Detail view is full screen
	if m.viewState == ViewDetail {
		return m.detail.View()
	}

	// Confirm dialog is full screen overlay
	if m.viewState == ViewConfirm && m.confirm != nil {
		return m.confirm.View()
	}

	// Selector dialog is full screen overlay
	if m.viewState == ViewSelector && m.selector != nil {
		return m.selector.View()
	}

	// Search tab input mode
	if m.viewState == ViewSearchTab {
		return m.searchInput.View()
	}

	var b strings.Builder

	// Header
	b.WriteString(m.header.View())
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.tabs.View())
	b.WriteString("\n")

	// Search tab: show command prompt info when on Search tab
	if m.currentTab == SearchTabIndex {
		if m.searchCommand != "" {
			b.WriteString(fmt.Sprintf("kubectl get %s", normalizeGetCommand(m.searchCommand)))
		} else {
			b.WriteString("Press [Enter] to enter a kubectl command")
		}
		b.WriteString("\n")
	}

	// Search bar (if active or filter confirmed)
	if m.search.IsActive() || m.search.IsFiltered() {
		b.WriteString(m.search.View())
		b.WriteString("\n")
	}

	// Show error if any
	if m.lastError != nil {
		b.WriteString("\nError: ")
		b.WriteString(m.lastError.Error())
		b.WriteString("\n")
	} else {
		// Resource list
		b.WriteString(m.list.View())
	}

	// Footer/help
	b.WriteString("\n\n")
	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...")
	} else if m.search.IsActive() {
		hint := "[Enter] confirm  [Esc] cancel"
		if m.search.HistoryLen() > 0 {
			hint += "  [↑↓] history"
		}
		if m.search.HasSuggestion() {
			hint += "  [Tab] → " + m.search.SuggestionFull()
		}
		b.WriteString(hint)
	} else if m.search.IsFiltered() {
		b.WriteString(wrapAtWidth("[Esc] clear filter  [/] modify filter  [d]escribe [L]ogs [D]elete [e]dit [x]exec  [w]save [+]new tab", m.width))
	} else if m.currentTab == SearchTabIndex {
		b.WriteString(wrapAtWidth("[Enter] enter command  [/]filter results  [r]efresh  [q]uit  [w]save as tab  [+]new tab", m.width))
	} else {
		b.WriteString(wrapAtWidth("[d]escribe [L]ogs [Y]aml [D]elete [e]dit [x]exec [R]estart  [c]ontext [n]amespace  [s]ort [/]search [r]efresh [?]help  [w]save [+]new tab", m.width))
	}

	return b.String()
}

// loadContext returns a command to load the current kubectl context
func (m *Model) loadContext() tea.Cmd {
	return func() tea.Msg {
		ctx, err := m.kubectl.GetCurrentContext()
		if err != nil {
			return ErrorMsg{Err: err}
		}

		ns, err := m.kubectl.GetCurrentNamespace()
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return ContextLoadedMsg{
			Context:   ctx,
			Namespace: ns,
		}
	}
}

// handleTabChange handles switching to a new tab
// saveCurrentTabSearch saves the active search filter for the current tab and
// deactivates the search component so the next tab starts clean.
func (m *Model) saveCurrentTabSearch() {
	if q := m.search.Query(); q != "" {
		m.tabSearchStates[m.currentTab] = q
	} else {
		delete(m.tabSearchStates, m.currentTab)
	}
	m.search.Deactivate()
}

func (m *Model) handleTabChange() tea.Cmd {
	// Search tab shows empty list until command is executed
	if m.currentTab == SearchTabIndex {
		m.resources = &kubectl.TableData{}
		m.list.SetItems([]string{}, [][]string{})
		return nil
	}
	// Reset list for the new tab so cursor starts at top
	m.list.SetItems([]string{}, [][]string{})
	return m.startLoading(m.loadResources())
}

// loadResources returns a command to load resources for the current tab
func (m *Model) loadResources() tea.Cmd {
	return func() tea.Msg {
		// Search tab: re-execute the last search command if one exists
		if m.currentTab == SearchTabIndex {
			if m.searchCommand != "" {
				output, err := m.kubectl.ExecuteRaw("get " + normalizeGetCommand(m.searchCommand))
				if err != nil {
					return ErrorMsg{Err: err}
				}
				data := kubectl.ParseTableOutput(output)
				return ResourcesLoadedMsg{Data: data}
			}
			return ResourcesLoadedMsg{Data: &kubectl.TableData{}}
		}

		// Config tabs are offset by 1 due to Search tab at index 0
		configTabIndex := m.currentTab - 1
		if configTabIndex < 0 || configTabIndex >= len(m.config.Tabs) {
			return ResourcesLoadedMsg{Data: &kubectl.TableData{}}
		}

		tab := m.config.Tabs[configTabIndex]
		output, err := m.kubectl.ExecuteRaw("get " + normalizeGetCommand(tab.Command))
		if err != nil {
			return ErrorMsg{Err: err}
		}

		data := kubectl.ParseTableOutput(output)
		return ResourcesLoadedMsg{Data: data}
	}
}

// executeSearchCommand executes the search tab command
func (m *Model) executeSearchCommand() tea.Cmd {
	return func() tea.Msg {
		if m.searchCommand == "" {
			return ResourcesLoadedMsg{Data: &kubectl.TableData{}}
		}

		output, err := m.kubectl.ExecuteRaw("get " + normalizeGetCommand(m.searchCommand))
		if err != nil {
			return ErrorMsg{Err: err}
		}

		data := kubectl.ParseTableOutput(output)
		return ResourcesLoadedMsg{Data: data}
	}
}

// loadResourceDetail returns a command to load detail for the selected resource
func (m *Model) loadResourceDetail(format detail.Format) tea.Cmd {
	return func() tea.Msg {
		selected := m.list.SelectedItem()
		if selected == nil || len(selected) == 0 {
			return ErrorMsg{Err: nil}
		}

		resourceType := m.getCurrentResourceType()
		if resourceType == "" {
			return ErrorMsg{Err: nil}
		}

		// Determine namespace and resource name from the selected row
		namespace := m.currentNamespace
		resourceName := selected[0]
		namespaceFromColumn := false

		// Check if there's a NAMESPACE column (for -A commands)
		if m.resources != nil {
			namespaceIdx := m.resources.GetColumnIndex("NAMESPACE")
			if namespaceIdx >= 0 && namespaceIdx < len(selected) {
				namespace = selected[namespaceIdx]
				namespaceFromColumn = true
			}
			// NAME column might be after NAMESPACE
			nameIdx := m.resources.GetColumnIndex("NAME")
			if nameIdx >= 0 && nameIdx < len(selected) {
				resourceName = selected[nameIdx]
			}
		}

		// No NAMESPACE column: fall back to the namespace in the tab command
		if !namespaceFromColumn {
			if cmdNs := getNamespaceFromCommand(m.getCurrentTabCommand()); cmdNs != "" {
				namespace = cmdNs
			}
		}

		var content string
		var err error

		switch format {
		case detail.FormatYAML:
			content, err = m.kubectl.GetResourceYAML(resourceType, resourceName, namespace)
		case detail.FormatJSON:
			content, err = m.kubectl.GetResourceJSON(resourceType, resourceName, namespace)
		default:
			// Table format - use describe
			content, _, err = m.kubectl.Execute("describe", resourceType, resourceName, "-n", namespace)
		}

		if err != nil {
			return ErrorMsg{Err: err}
		}

		m.viewState = ViewDetail
		m.detail.SetSize(m.width, m.height-2)

		return DetailLoadedMsg{
			ResourceName: resourceName,
			Content:      content,
			Format:       int(format),
		}
	}
}

// getSelectedResourceInfo returns resource names and namespace for selected items
func (m *Model) getSelectedResourceInfo() (names []string, namespace string) {
	namespace = m.currentNamespace

	// Get selected items (or current item if none selected)
	items := m.list.SelectedItems()
	if items == nil {
		return nil, namespace
	}

	// Extract names and potentially namespace from items
	nameIdx := 0
	namespaceIdx := -1

	if m.resources != nil {
		// Check for NAMESPACE column (present in -A commands)
		namespaceIdx = m.resources.GetColumnIndex("NAMESPACE")
		// NAME column might be after NAMESPACE
		nameColIdx := m.resources.GetColumnIndex("NAME")
		if nameColIdx >= 0 {
			nameIdx = nameColIdx
		}
	}

	namespaceExtracted := false
	for _, item := range items {
		if len(item) > nameIdx {
			names = append(names, item[nameIdx])
		}
		// Use namespace from the first item only when a NAMESPACE column is present
		if !namespaceExtracted && namespaceIdx >= 0 && namespaceIdx < len(item) {
			namespace = item[namespaceIdx]
			namespaceExtracted = true
		}
	}

	// If no NAMESPACE column in the data, fall back to the namespace specified
	// in the current tab command (e.g. "get pods -n rhwa" → "rhwa")
	if !namespaceExtracted {
		if cmdNs := getNamespaceFromCommand(m.getCurrentTabCommand()); cmdNs != "" {
			namespace = cmdNs
		}
	}

	return names, namespace
}

// describeResources returns a command to describe selected resources
func (m *Model) describeResources() tea.Cmd {
	return func() tea.Msg {
		names, namespace := m.getSelectedResourceInfo()
		if len(names) == 0 {
			return ErrorMsg{Err: nil}
		}

		resourceType := m.getCurrentResourceType()
		if resourceType == "" {
			return ErrorMsg{Err: nil}
		}

		ctx := actions.NewContext(m.kubectl, resourceType, names, namespace)

		content, err := actions.Describe(ctx)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return DescribeLoadedMsg{
			ResourceNames: names,
			Content:       content,
		}
	}
}

// isPodResource returns true if the resource type is a pod
func isPodResource(resourceType string) bool {
	return resourceType == "pods" || resourceType == "pod" || resourceType == "po"
}

// viewLogs returns a command to view logs for the selected pod.
// It always fetches containers and shows a selector popup with
// "log <container>" and "log <container> --follow" entries.
func (m *Model) viewLogs() tea.Cmd {
	return func() tea.Msg {
		resourceType := m.getCurrentResourceType()

		// Logs only work for pods
		if !isPodResource(resourceType) {
			return ErrorMsg{Err: nil}
		}

		names, namespace := m.getSelectedResourceInfo()
		if len(names) == 0 {
			return ErrorMsg{Err: nil}
		}

		// For logs, we only use the first selected pod
		podName := names[0]
		ctx := actions.NewContext(m.kubectl, "pod", []string{podName}, namespace)

		// Get container list
		containers, err := actions.GetContainers(ctx)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return ContainersLoadedMsg{
			PodName:    podName,
			Namespace:  namespace,
			Containers: containers,
		}
	}
}

// confirmDelete shows a confirmation dialog for delete
func (m *Model) confirmDelete() tea.Cmd {
	return func() tea.Msg {
		names, namespace := m.getSelectedResourceInfo()
		if len(names) == 0 {
			return nil
		}

		resourceType := m.getCurrentResourceType()
		if resourceType == "" {
			return nil
		}

		var message string
		if len(names) == 1 {
			message = fmt.Sprintf("Delete %s/%s?\n\nThis action cannot be undone.", resourceType, names[0])
		} else {
			message = fmt.Sprintf("Delete %d %s resources?\n\nThis action cannot be undone.", len(names), resourceType)
		}

		m.confirm = dialog.NewConfirm("Confirm Delete", message)
		m.confirm.SetSize(m.width, m.height)
		m.pendingAction = "delete"
		m.pendingNames = names
		m.pendingNs = namespace
		m.viewState = ViewConfirm

		return nil
	}
}

// executeDelete performs the actual delete operation
func (m *Model) executeDelete() tea.Cmd {
	return func() tea.Msg {
		if len(m.pendingNames) == 0 {
			return nil
		}

		resourceType := m.getCurrentResourceType()
		if resourceType == "" {
			return nil
		}

		ctx := actions.NewContext(m.kubectl, resourceType, m.pendingNames, m.pendingNs)

		results, err := actions.Delete(ctx)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Clear pending state
		m.pendingAction = ""
		m.pendingNames = nil
		m.pendingNs = ""

		// Build result message
		var messages []string
		for _, r := range results {
			messages = append(messages, r.Message)
		}

		// Show result in detail view briefly, then refresh
		m.detail.SetContent("Delete Result", strings.Join(messages, "\n"), detail.FormatTable)
		m.detail.SetSize(m.width, m.height-2)
		m.viewState = ViewDetail

		return nil
	}
}

// editResource opens the resource in an editor
func (m *Model) editResource() tea.Cmd {
	names, namespace := m.getSelectedResourceInfo()
	if len(names) == 0 {
		return nil
	}

	resourceType := m.getCurrentResourceType()
	if resourceType == "" {
		return nil
	}

	// Return a command that suspends the TUI
	return tea.ExecProcess(
		makeKubectlEditCmd(m.kubectl.BinaryPath(), resourceType, names[0], namespace),
		func(err error) tea.Msg {
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return RefreshMsg{}
		},
	)
}

// makeKubectlEditCmd creates an exec.Cmd for kubectl edit
func makeKubectlEditCmd(kubectlBin, resource, name, namespace string) *exec.Cmd {
	args := []string{"edit", resource, name}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	cmd := exec.Command(kubectlBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// execIntoPod starts an exec session in the pod
func (m *Model) execIntoPod() tea.Cmd {
	names, namespace := m.getSelectedResourceInfo()
	if len(names) == 0 {
		return nil
	}

	resourceType := m.getCurrentResourceType()
	// Exec only works for pods
	if !isPodResource(resourceType) {
		return nil
	}

	// Return a command that suspends the TUI
	return tea.ExecProcess(
		makeKubectlExecCmd(m.kubectl.BinaryPath(), names[0], namespace, ""),
		func(err error) tea.Msg {
			return nil // Don't show error, just return to TUI
		},
	)
}

// makeKubectlExecCmd creates an exec.Cmd for kubectl exec
func makeKubectlExecCmd(kubectlBin, podName, namespace, container string) *exec.Cmd {
	args := []string{"exec", "-it", podName}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--", "/bin/sh")
	cmd := exec.Command(kubectlBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// launchInNewShell opens a command in a new terminal window.
// It tries common terminal emulators in order of preference.
func launchInNewShell(args []string) {
	shellCmd := strings.Join(args, " ")

	// Try common terminal emulators
	terminals := []struct {
		bin  string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", "sh", "-c", shellCmd}},
		{"gnome-terminal", []string{"--", "sh", "-c", shellCmd}},
		{"konsole", []string{"-e", "sh", "-c", shellCmd}},
		{"xfce4-terminal", []string{"-e", shellCmd}},
		{"xterm", []string{"-e", shellCmd}},
		{"open", []string{"-a", "Terminal", "--args", shellCmd}}, // macOS
	}

	for _, t := range terminals {
		if path, err := exec.LookPath(t.bin); err == nil {
			cmd := exec.Command(path, t.args...)
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.Stdin = nil
			_ = cmd.Start()
			return
		}
	}
}

// makeKubectlLogsFollowCmd creates an exec.Cmd for kubectl logs -f
func makeKubectlLogsFollowCmd(kubectlBin, podName, namespace, container string) *exec.Cmd {
	args := []string{"logs", "-f", podName}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--tail", "500")
	cmd := exec.Command(kubectlBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// applyTabSort sorts data according to the effective sort state for the current
// tab.  Runtime sort (set via 's') takes precedence over config-level sort.
func (m *Model) applyTabSort(data *kubectl.TableData) {
	if data == nil {
		return
	}

	// Runtime sort overrides config sort
	if rs, ok := m.tabRuntimeSorts[m.currentTab]; ok && rs.SortBy != "" {
		data.Sort(rs.SortBy, rs.SortReverse)
		return
	}

	// Fall back to config-level sort (config tabs only)
	if m.currentTab != SearchTabIndex {
		configIdx := m.currentTab - 1
		if configIdx >= 0 && configIdx < len(m.config.Tabs) {
			tab := m.config.Tabs[configIdx]
			if tab.SortBy != "" {
				data.Sort(tab.SortBy, tab.SortReverse)
			}
		}
	}
}

// openSortSelector builds sort options from the current tab's headers and opens
// the selector dialog.
func (m *Model) openSortSelector() {
	if m.resources == nil || len(m.resources.Headers) == 0 {
		return
	}

	currentSort := m.tabRuntimeSorts[m.currentTab]
	// Fall back to config sort as the "current" sort when no runtime override exists.
	if currentSort.SortBy == "" && m.currentTab != SearchTabIndex {
		configIdx := m.currentTab - 1
		if configIdx >= 0 && configIdx < len(m.config.Tabs) {
			tab := m.config.Tabs[configIdx]
			currentSort = TabSortState{SortBy: tab.SortBy, SortReverse: tab.SortReverse}
		}
	}

	isActive := func(by string, rev bool) bool {
		return strings.EqualFold(currentSort.SortBy, by) && currentSort.SortReverse == rev
	}
	prefix := func(by string, rev bool) string {
		if isActive(by, rev) {
			return "● "
		}
		return "  "
	}

	optionMap := make(map[string]TabSortState)
	var items []string

	addOption := func(label, sortBy string, rev bool) {
		optionMap[label] = TabSortState{SortBy: sortBy, SortReverse: rev}
		items = append(items, label)
	}

	// Creation time is always available (uses AGE column when present).
	hasAge := m.resources.GetColumnIndex("AGE") >= 0
	if hasAge {
		addOption(prefix("creation_time", false)+"creation_time · newest first", "creation_time", false)
		addOption(prefix("creation_time", true)+"creation_time · oldest first", "creation_time", true)
	}

	// One ascending + descending entry per column (skip AGE – covered above).
	for _, h := range m.resources.Headers {
		if strings.EqualFold(h, "AGE") {
			continue
		}
		addOption(prefix(h, false)+h+" · A→Z", h, false)
		addOption(prefix(h, true)+h+" · Z→A", h, true)
	}

	// If no AGE column but creation_time was set, still offer it.
	if !hasAge {
		addOption(prefix("creation_time", false)+"creation_time · newest first", "creation_time", false)
		addOption(prefix("creation_time", true)+"creation_time · oldest first", "creation_time", true)
	}

	m.sortOptionMap = optionMap
	m.selector = dialog.NewSelector("Sort by", items)
	m.selector.SetSize(m.width, m.height)
	m.pendingAction = "sort"
	m.viewState = ViewSelector
}

// rolloutRestart triggers a rollout restart
func (m *Model) rolloutRestart() tea.Cmd {
	return func() tea.Msg {
		names, namespace := m.getSelectedResourceInfo()
		if len(names) == 0 {
			return nil
		}

		resourceType := m.getCurrentResourceType()
		if resourceType == "" {
			return nil
		}

		ctx := actions.NewContext(m.kubectl, resourceType, names, namespace)

		output, err := actions.RolloutRestart(ctx)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Show result
		m.detail.SetContent("Rollout Restart", output, detail.FormatTable)
		m.detail.SetSize(m.width, m.height-2)
		m.viewState = ViewDetail

		return nil
	}
}

// handleSelectorResult processes the selector dialog result
func (m *Model) handleSelectorResult() tea.Cmd {
	if m.selector == nil {
		return nil
	}

	selected := m.selector.SelectedItem()
	action := m.pendingAction

	m.pendingAction = ""
	m.selector = nil

	switch action {
	case "sort":
		if state, ok := m.sortOptionMap[selected]; ok {
			m.tabRuntimeSorts[m.currentTab] = state
			if m.resources != nil {
				m.applyTabSort(m.resources)
				if m.search.IsFiltered() {
					m.search.SetItems(m.resources.Rows)
					m.list.UpdateItems(m.resources.Headers, m.search.FilteredItems())
				} else {
					m.list.UpdateItems(m.resources.Headers, m.resources.Rows)
				}
			}
		}
		return nil
	case "log-options":
		// Parse "log <container>" or "log <container> --follow"
		follow := strings.HasSuffix(selected, " --follow")
		container := strings.TrimPrefix(selected, "log ")
		container = strings.TrimSuffix(container, " --follow")
		return m.viewLogsWithContainer(container, follow)
	case "container-exec":
		// Use selected container for exec
		return m.execWithContainer(selected)
	case "context":
		// Switch context
		return m.switchContext(selected)
	case "namespace":
		// Switch namespace
		return m.switchNamespace(selected)
	}

	return nil
}

// viewLogsWithContainer views logs for a specific container
func (m *Model) viewLogsWithContainer(container string, follow bool) tea.Cmd {
	newShell := m.config.LogsFollowInNewShell
	return func() tea.Msg {
		if len(m.pendingNames) == 0 {
			return nil
		}

		podName := m.pendingNames[0]
		namespace := m.pendingNs

		m.pendingNames = nil
		m.pendingNs = ""

		if follow {
			return LogsFollowMsg{
				PodName:   podName,
				Namespace: namespace,
				Container: container,
				NewShell:  newShell,
			}
		}

		ctx := actions.NewContext(m.kubectl, "pod", []string{podName}, namespace)

		opts := actions.LogsOptions{
			Container: container,
			TailLines: 500,
		}

		content, err := actions.Logs(ctx, opts)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return LogsLoadedMsg{
			PodName:   podName,
			Container: container,
			Content:   content,
		}
	}
}

// execWithContainer starts exec with a specific container
func (m *Model) execWithContainer(container string) tea.Cmd {
	if len(m.pendingNames) == 0 {
		return nil
	}

	podName := m.pendingNames[0]
	namespace := m.pendingNs
	kubectlBin := m.kubectl.BinaryPath()

	m.pendingNames = nil
	m.pendingNs = ""

	return tea.ExecProcess(
		makeKubectlExecCmd(kubectlBin, podName, namespace, container),
		func(err error) tea.Msg {
			return nil
		},
	)
}

// switchContext switches to a different kubectl context
func (m *Model) switchContext(contextName string) tea.Cmd {
	return func() tea.Msg {
		_, _, err := m.kubectl.Execute("config", "use-context", contextName)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		// Reload context and resources
		return RefreshMsg{}
	}
}

// switchNamespace updates the current namespace
func (m *Model) switchNamespace(namespace string) tea.Cmd {
	return func() tea.Msg {
		m.currentNamespace = namespace
		m.header.SetNamespace(namespace)
		// Reload resources with new namespace
		return RefreshMsg{}
	}
}

// showContextSelector displays a selector for available contexts
func (m *Model) showContextSelector() tea.Cmd {
	return func() tea.Msg {
		contexts, err := m.kubectl.GetContexts()
		if err != nil {
			return ErrorMsg{Err: err}
		}

		if len(contexts) == 0 {
			return nil
		}

		m.selector = dialog.NewSelector("Select Context", contexts)
		m.selector.SetSize(m.width, m.height)
		m.pendingAction = "context"
		m.viewState = ViewSelector

		return nil
	}
}

// addNewTab validates name+command, writes the updated config to disk, and
// returns a TabAddedMsg so the main Update() can update in-memory state.
func (m *Model) addNewTab(name, command string) tea.Cmd {
	name = strings.TrimSpace(name)
	command = normalizeGetCommand(command)
	configPath := m.configPath

	existingTabs := make([]config.TabConfig, len(m.config.Tabs))
	copy(existingTabs, m.config.Tabs)
	cfgCopy := *m.config
	cfgCopy.Tabs = existingTabs

	return func() tea.Msg {
		if name == "" {
			return ErrorMsg{Err: fmt.Errorf("tab name cannot be empty")}
		}
		if command == "" {
			return ErrorMsg{Err: fmt.Errorf("command cannot be empty")}
		}

		cfgCopy.Tabs = append(cfgCopy.Tabs, config.TabConfig{Name: name, Command: command})

		if err := cfgCopy.Validate(); err != nil {
			return ErrorMsg{Err: fmt.Errorf("invalid config: %w", err)}
		}

		if configPath != "" {
			if err := config.SaveConfig(&cfgCopy, configPath); err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to save config: %w", err)}
			}
		}

		return TabAddedMsg{Name: name, Command: command}
	}
}

// saveCurrentTabToConfig persists the current tab's command to disk.
func (m *Model) saveCurrentTabToConfig() tea.Cmd {
	configTabIndex := m.currentTab - 1
	if configTabIndex < 0 || configTabIndex >= len(m.config.Tabs) {
		return nil
	}
	configPath := m.configPath
	if configPath == "" {
		return nil
	}

	existingTabs := make([]config.TabConfig, len(m.config.Tabs))
	copy(existingTabs, m.config.Tabs)
	cfgCopy := *m.config
	cfgCopy.Tabs = existingTabs

	return func() tea.Msg {
		if err := config.SaveConfig(&cfgCopy, configPath); err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to save config: %w", err)}
		}
		return ConfigSavedMsg{}
	}
}

// saveSearchAsTab adds the current Search-tab command as a new named tab and
// saves to disk.
func (m *Model) saveSearchAsTab(name string) tea.Cmd {
	name = strings.TrimSpace(name)
	command := normalizeGetCommand(m.searchCommand)
	configPath := m.configPath

	existingTabs := make([]config.TabConfig, len(m.config.Tabs))
	copy(existingTabs, m.config.Tabs)
	cfgCopy := *m.config
	cfgCopy.Tabs = existingTabs

	return func() tea.Msg {
		if name == "" {
			return ErrorMsg{Err: fmt.Errorf("tab name cannot be empty")}
		}
		if command == "" {
			return ErrorMsg{Err: fmt.Errorf("no command to save")}
		}

		cfgCopy.Tabs = append(cfgCopy.Tabs, config.TabConfig{Name: name, Command: command})

		if err := cfgCopy.Validate(); err != nil {
			return ErrorMsg{Err: fmt.Errorf("invalid config: %w", err)}
		}

		if configPath != "" {
			if err := config.SaveConfig(&cfgCopy, configPath); err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to save config: %w", err)}
			}
		}

		return TabAddedMsg{Name: name, Command: command}
	}
}

// showNamespaceSelector displays a selector for available namespaces
func (m *Model) showNamespaceSelector() tea.Cmd {
	return func() tea.Msg {
		namespaces, err := m.kubectl.GetNamespaces()
		if err != nil {
			return ErrorMsg{Err: err}
		}

		if len(namespaces) == 0 {
			return nil
		}

		m.selector = dialog.NewSelector("Select Namespace", namespaces)
		m.selector.SetSize(m.width, m.height)
		m.pendingAction = "namespace"
		m.viewState = ViewSelector

		return nil
	}
}
