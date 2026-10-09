package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aymanbagabas/go-osc52/v2"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	accDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	accPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	bucketApp "github.com/sekai-labs/kumokura/internal/buckets/application"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objApp "github.com/sekai-labs/kumokura/internal/objects/application"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	objPorts "github.com/sekai-labs/kumokura/internal/objects/ports"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/components"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/keymap"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/messages"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/views"
	syncPorts "github.com/sekai-labs/kumokura/internal/synchronization/ports"
	transferApp "github.com/sekai-labs/kumokura/internal/transfers/application"
	transferDomain "github.com/sekai-labs/kumokura/internal/transfers/domain"
)

type Services struct {
	AccountService  accPorts.AccountService
	BucketService   *bucketApp.BucketService
	ObjectService   *objApp.ObjectService
	TransferService *transferApp.TransferService
	SyncService     syncPorts.SyncEngine
	SyncRepo        syncPorts.SyncRepository
}

type Model struct {
	width  int
	height int

	services Services
	styles   styles.Styles
	keymap   keymap.KeyMap

	tabBar    components.TabBar
	statusBar components.StatusBar
	searchBar components.SearchBar

	activeTab int

	explorerView  views.ExplorerView
	transfersView views.TransfersView
	syncView      views.SyncView
	helpView      views.HelpView

	activeAccount string
	activeRegion  string
	activeBucket  string

	showHelpModal   bool
	showDeleteModal bool
	deleteTargetKey string
	uploadModal     components.UploadModal
	downloadModal   components.DownloadModal
	presignModal    components.PresignModal

	notification string
}

func NewModel(services Services) Model {
	st := styles.NewStyles(styles.DarkTheme)
	tabs := []components.TabItem{
		{ID: "buckets", Title: "[1 Buckets]"},
		{ID: "objects", Title: "[2 Objects]"},
		{ID: "transfers", Title: "[3 Transfers]"},
		{ID: "sync", Title: "[4 Sync]"},
	}

	return Model{
		width:         120,
		height:        40,
		services:      services,
		styles:        st,
		keymap:        keymap.DefaultKeyMap,
		tabBar:        components.NewTabBar(tabs, st),
		statusBar:     components.NewStatusBar(st),
		searchBar:     components.NewSearchBar(st),
		uploadModal:   components.NewUploadModal(st),
		downloadModal: components.NewDownloadModal(st),
		presignModal:  components.NewPresignModal(st),
		activeTab:     0,
		explorerView:  views.NewExplorerView(st),
		transfersView: views.NewTransfersView(st),
		syncView:      views.NewSyncView(st),
		helpView:      views.NewHelpView(st),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadAccountsCmd(),
		m.loadBucketsCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case messages.AccountsLoadedMsg:
		if msg.Err == nil && len(msg.Accounts) > 0 {
			m.activeAccount = msg.Accounts[0].Name
			m.activeRegion = msg.Accounts[0].Region
		}

	case messages.BucketsLoadedMsg:
		if msg.Err == nil {
			m.explorerView.Buckets = msg.Buckets
			if len(msg.Buckets) > 0 && m.activeBucket == "" {
				m.activeBucket = msg.Buckets[0].Name
				m.explorerView.ActiveBucket = m.activeBucket
				cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, ""))
			}
		}

	case messages.ObjectsLoadedMsg:
		if msg.Err == nil {
			m.explorerView.Objects = msg.Result.Objects
			m.explorerView.Prefixes = msg.Result.CommonPrefixes
			m.explorerView.SelectedObject = 0
			if len(msg.Result.Objects) > 0 {
				cmds = append(cmds, m.loadObjectMetadataCmd(m.activeBucket, msg.Result.Objects[0].Key))
			}
		}

	case messages.ObjectMetadataLoadedMsg:
		if msg.Err == nil {
			m.explorerView.PreviewMetadata = &msg.Metadata
			m.explorerView.PreviewTags = msg.Tags
		}

	case messages.ContentPreviewLoadedMsg:
		if msg.Err == nil {
			m.explorerView.PreviewContent = []byte(msg.Content)
		}

	case messages.UploadFinishedMsg:
		if msg.Err != nil {
			m.notification = fmt.Sprintf("Upload failed: %v", msg.Err)
		} else {
			m.notification = fmt.Sprintf("Successfully uploaded %s", msg.Key)
			cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, m.explorerView.CurrentPrefix))
		}

	case messages.FolderUploadFinishedMsg:
		if msg.Err != nil {
			m.notification = fmt.Sprintf("Folder upload failed: %v", msg.Err)
		} else {
			m.notification = fmt.Sprintf("Uploaded %d objects (%s) to %s", msg.TotalCount, formatBytes(msg.TotalBytes), msg.Prefix)
			cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, m.explorerView.CurrentPrefix))
		}

	case messages.DownloadFinishedMsg:
		if msg.Err != nil {
			m.notification = fmt.Sprintf("Download failed: %v", msg.Err)
		} else {
			m.notification = fmt.Sprintf("Downloaded %s to %s", msg.Key, msg.Path)
		}

	case messages.FolderDownloadFinishedMsg:
		if msg.Err != nil {
			m.notification = fmt.Sprintf("Folder download failed: %v", msg.Err)
		} else {
			m.notification = fmt.Sprintf("Downloaded %d objects (%s) to %s", msg.TotalCount, formatBytes(msg.TotalBytes), msg.LocalDest)
		}

	case messages.PresignedURLGeneratedMsg:
		if msg.Err != nil {
			m.presignModal.ErrorText = fmt.Sprintf("Presign failed: %v", msg.Err)
		} else {
			m.presignModal.GeneratedURL = msg.URL
			m.presignModal.Copied = msg.Copied
			if msg.Copied {
				m.notification = "Presigned URL copied to clipboard"
			}
		}

	case messages.TransfersLoadedMsg:
		if msg.Err == nil {
			m.transfersView.Jobs = msg.Jobs
		}

	case messages.SyncJobsLoadedMsg:
		if msg.Err == nil {
			m.syncView.Jobs = msg.Jobs
		}

	case messages.StatusNotificationMsg:
		m.notification = msg.Message
	case tea.KeyMsg:
		if m.showHelpModal {
			if key.Matches(msg, m.keymap.Escape, m.keymap.Help, m.keymap.Quit) {
				m.showHelpModal = false
			}
			return m, nil
		}

		if m.showDeleteModal {
			switch msg.String() {
			case "y", "Y", "enter":
				m.showDeleteModal = false
				cmds = append(cmds, m.deleteObjectCmd(m.activeBucket, m.deleteTargetKey))
				return m, tea.Batch(cmds...)
			case "n", "N", "esc":
				m.showDeleteModal = false
			}
			return m, nil
		}

		if m.uploadModal.Active {
			switch {
			case key.Matches(msg, m.keymap.Escape):
				m.uploadModal.Active = false
				m.uploadModal.Input.Blur()
			case key.Matches(msg, m.keymap.Enter):
				path := strings.TrimSpace(m.uploadModal.Input.Value())
				if path == "" {
					m.uploadModal.ErrorText = "Please specify a valid file or folder path"
				} else {
					fi, err := os.Stat(path)
					if err != nil {
						m.uploadModal.ErrorText = fmt.Sprintf("Path not found: %s", filepath.Base(path))
					} else if fi.IsDir() {
						m.uploadModal.Active = false
						m.uploadModal.Input.Blur()
						m.notification = fmt.Sprintf("Uploading folder %s...", filepath.Base(path))
						cmds = append(cmds, m.uploadFolderCmd(m.activeBucket, m.explorerView.CurrentPrefix, path))
					} else {
						m.uploadModal.Active = false
						m.uploadModal.Input.Blur()
						targetKey := m.uploadModal.TargetKey
						if targetKey == "" {
							targetKey = filepath.Base(path)
						}
						m.notification = fmt.Sprintf("Uploading %s...", filepath.Base(path))
						cmds = append(cmds, m.uploadObjectCmd(m.activeBucket, targetKey, path))
					}
				}
			default:
				var tiCmd tea.Cmd
				m.uploadModal.Input, tiCmd = m.uploadModal.Input.Update(msg)
				cmds = append(cmds, tiCmd)
			}
			return m, tea.Batch(cmds...)
		}

		if m.downloadModal.Active {
			switch {
			case key.Matches(msg, m.keymap.Escape):
				m.downloadModal.Active = false
				m.downloadModal.Input.Blur()
			case key.Matches(msg, m.keymap.Enter):
				destDir := strings.TrimSpace(m.downloadModal.Input.Value())
				if destDir == "" {
					destDir = "."
				}
				m.downloadModal.Active = false
				m.downloadModal.Input.Blur()

				if m.downloadModal.IsFolder {
					m.notification = fmt.Sprintf("Downloading folder %s to %s...", m.downloadModal.TargetName, destDir)
					cmds = append(cmds, m.downloadFolderCmd(m.activeBucket, m.downloadModal.TargetName, destDir))
				} else {
					m.notification = fmt.Sprintf("Downloading %s to %s...", filepath.Base(m.downloadModal.TargetName), destDir)
					cmds = append(cmds, m.downloadObjectCmd(m.activeBucket, m.downloadModal.TargetName, destDir))
				}
			default:
				var tiCmd tea.Cmd
				m.downloadModal.Input, tiCmd = m.downloadModal.Input.Update(msg)
				cmds = append(cmds, tiCmd)
			}
			return m, tea.Batch(cmds...)
		}

		if m.presignModal.Active {
			switch {
			case key.Matches(msg, m.keymap.Escape) || (m.presignModal.GeneratedURL != "" && key.Matches(msg, m.keymap.Enter)):
				m.presignModal.Active = false
				m.presignModal.Input.Blur()
			case key.Matches(msg, m.keymap.Enter):
				durStr := strings.TrimSpace(m.presignModal.Input.Value())
				if durStr == "" {
					durStr = "60m"
				}
				dur, err := time.ParseDuration(durStr)
				if err != nil {
					m.presignModal.ErrorText = "Invalid duration (e.g. 15m, 1h, 24h)"
				} else {
					cmds = append(cmds, m.generatePresignedURLCmd(m.activeBucket, m.presignModal.ObjectKey, dur))
				}
			default:
				var tiCmd tea.Cmd
				m.presignModal.Input, tiCmd = m.presignModal.Input.Update(msg)
				cmds = append(cmds, tiCmd)
			}
			return m, tea.Batch(cmds...)
		}

		if m.searchBar.Active {
			switch {
			case key.Matches(msg, m.keymap.Escape):
				m.searchBar.Active = false
				m.searchBar.Input.Blur()
			case key.Matches(msg, m.keymap.Enter):
				m.searchBar.Active = false
				m.searchBar.Input.Blur()
				cmds = append(cmds, m.filterObjectsCmd(m.searchBar.Input.Value()))
			default:
				var tiCmd tea.Cmd
				m.searchBar.Input, tiCmd = m.searchBar.Input.Update(msg)
				cmds = append(cmds, tiCmd)
			}
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Help):
			m.showHelpModal = true
			return m, nil

		case key.Matches(msg, m.keymap.Filter):
			m.searchBar.Active = true
			m.searchBar.Input.Focus()
			return m, nil

		case key.Matches(msg, m.keymap.Tab):
			m.explorerView.ActivePaneIndex = (m.explorerView.ActivePaneIndex + 1) % 3

		case key.Matches(msg, m.keymap.ShiftTab):
			m.explorerView.ActivePaneIndex = (m.explorerView.ActivePaneIndex + 2) % 3

		case key.Matches(msg, m.keymap.Inspector), key.Matches(msg, m.keymap.ToggleDetail):
			m.explorerView.ShowPreview = !m.explorerView.ShowPreview

		case key.Matches(msg, m.keymap.Refresh):
			cmds = append(cmds, m.loadBucketsCmd())
			if m.activeBucket != "" {
				cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, m.explorerView.CurrentPrefix))
			}
			if m.activeTab == 2 {
				cmds = append(cmds, m.loadTransfersCmd())
			} else if m.activeTab == 3 {
				cmds = append(cmds, m.loadSyncJobsCmd())
			}

		case msg.String() == "1":
			m.activeTab = 0
			m.tabBar.Active = 0
			m.explorerView.ActivePaneIndex = 0
		case msg.String() == "2":
			m.activeTab = 1
			m.tabBar.Active = 1
			m.explorerView.ActivePaneIndex = 1
		case msg.String() == "3":
			m.activeTab = 2
			m.tabBar.Active = 2
			cmds = append(cmds, m.loadTransfersCmd())
		case msg.String() == "4":
			m.activeTab = 3
			m.tabBar.Active = 3
			cmds = append(cmds, m.loadSyncJobsCmd())
		default:
			if m.activeTab == 0 {
				m, cmds = m.handleExplorerKeys(msg, cmds)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleExplorerKeys(msg tea.KeyMsg, cmds []tea.Cmd) (Model, []tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Up):
		if m.explorerView.ActivePaneIndex == 0 {
			if m.explorerView.SelectedBucket > 0 {
				m.explorerView.SelectedBucket--
			}
		} else if m.explorerView.ActivePaneIndex == 1 {
			if m.explorerView.SelectedObject > 0 {
				m.explorerView.SelectedObject--
				cmds = append(cmds, m.inspectCurrentObjectCmd())
			}
		}

	case key.Matches(msg, m.keymap.Down):
		if m.explorerView.ActivePaneIndex == 0 {
			if m.explorerView.SelectedBucket < len(m.explorerView.Buckets)-1 {
				m.explorerView.SelectedBucket++
			}
		} else if m.explorerView.ActivePaneIndex == 1 {
			total := len(m.explorerView.Prefixes) + len(m.explorerView.Objects)
			if m.explorerView.SelectedObject < total-1 {
				m.explorerView.SelectedObject++
				cmds = append(cmds, m.inspectCurrentObjectCmd())
			}
		}

	case key.Matches(msg, m.keymap.PageUp):
		if m.explorerView.ActivePaneIndex == 0 {
			m.explorerView.SelectedBucket = max(0, m.explorerView.SelectedBucket-10)
		} else if m.explorerView.ActivePaneIndex == 1 {
			m.explorerView.SelectedObject = max(0, m.explorerView.SelectedObject-10)
			cmds = append(cmds, m.inspectCurrentObjectCmd())
		}

	case key.Matches(msg, m.keymap.PageDown):
		if m.explorerView.ActivePaneIndex == 0 {
			m.explorerView.SelectedBucket = min(max(0, len(m.explorerView.Buckets)-1), m.explorerView.SelectedBucket+10)
		} else if m.explorerView.ActivePaneIndex == 1 {
			total := len(m.explorerView.Prefixes) + len(m.explorerView.Objects)
			m.explorerView.SelectedObject = min(max(0, total-1), m.explorerView.SelectedObject+10)
			cmds = append(cmds, m.inspectCurrentObjectCmd())
		}

	case key.Matches(msg, m.keymap.Top):
		if m.explorerView.ActivePaneIndex == 0 {
			m.explorerView.SelectedBucket = 0
		} else if m.explorerView.ActivePaneIndex == 1 {
			m.explorerView.SelectedObject = 0
			cmds = append(cmds, m.inspectCurrentObjectCmd())
		}

	case key.Matches(msg, m.keymap.Bottom):
		if m.explorerView.ActivePaneIndex == 0 {
			m.explorerView.SelectedBucket = max(0, len(m.explorerView.Buckets)-1)
		} else if m.explorerView.ActivePaneIndex == 1 {
			total := len(m.explorerView.Prefixes) + len(m.explorerView.Objects)
			m.explorerView.SelectedObject = max(0, total-1)
			cmds = append(cmds, m.inspectCurrentObjectCmd())
		}

	case key.Matches(msg, m.keymap.Enter):
		if m.explorerView.ActivePaneIndex == 0 {
			if len(m.explorerView.Buckets) > m.explorerView.SelectedBucket {
				m.activeBucket = m.explorerView.Buckets[m.explorerView.SelectedBucket].Name
				m.explorerView.ActiveBucket = m.activeBucket
				m.explorerView.CurrentPrefix = ""
				m.explorerView.ActivePaneIndex = 1
				cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, ""))
			}
		} else if m.explorerView.ActivePaneIndex == 1 {
			if m.explorerView.SelectedObject < len(m.explorerView.Prefixes) {
				targetPrefix := m.explorerView.Prefixes[m.explorerView.SelectedObject].Prefix
				m.explorerView.CurrentPrefix = targetPrefix
				cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, targetPrefix))
			}
		}

	case key.Matches(msg, m.keymap.Back):
		if m.explorerView.CurrentPrefix != "" {
			trimmed := strings.TrimSuffix(m.explorerView.CurrentPrefix, "/")
			lastSlash := strings.LastIndex(trimmed, "/")
			if lastSlash >= 0 {
				m.explorerView.CurrentPrefix = trimmed[:lastSlash+1]
			} else {
				m.explorerView.CurrentPrefix = ""
			}
			cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, m.explorerView.CurrentPrefix))
		}

	case key.Matches(msg, m.keymap.Delete):
		prefixLen := len(m.explorerView.Prefixes)
		if m.explorerView.SelectedObject >= prefixLen {
			objIdx := m.explorerView.SelectedObject - prefixLen
			if objIdx < len(m.explorerView.Objects) {
				m.deleteTargetKey = m.explorerView.Objects[objIdx].Key
				m.showDeleteModal = true
			}
		}

	case key.Matches(msg, m.keymap.Upload):
		if m.activeBucket == "" {
			m.notification = "Select a bucket before uploading"
		} else {
			m.uploadModal.Active = true
			m.uploadModal.Destination = fmt.Sprintf("s3://%s/%s", m.activeBucket, m.explorerView.CurrentPrefix)
			m.uploadModal.ErrorText = ""
			m.uploadModal.Input.SetValue("")
			m.uploadModal.Input.Focus()
		}

	case key.Matches(msg, m.keymap.Download):
		if m.activeBucket == "" {
			m.notification = "Select a bucket before downloading"
		} else {
			prefixLen := len(m.explorerView.Prefixes)
			if m.explorerView.SelectedObject < prefixLen {
				targetFolder := m.explorerView.Prefixes[m.explorerView.SelectedObject].Prefix
				m.downloadModal.Active = true
				m.downloadModal.TargetName = targetFolder
				m.downloadModal.IsFolder = true
				m.downloadModal.ErrorText = ""
				m.downloadModal.Input.SetValue("./")
				m.downloadModal.Input.Focus()
			} else {
				objIdx := m.explorerView.SelectedObject - prefixLen
				if objIdx < len(m.explorerView.Objects) {
					targetKey := m.explorerView.Objects[objIdx].Key
					m.downloadModal.Active = true
					m.downloadModal.TargetName = targetKey
					m.downloadModal.IsFolder = false
					m.downloadModal.ErrorText = ""
					m.downloadModal.Input.SetValue("./")
					m.downloadModal.Input.Focus()
				}
			}
		}

	case key.Matches(msg, m.keymap.Presign):
		if m.activeBucket == "" {
			m.notification = "Select a bucket before generating presigned URL"
		} else {
			prefixLen := len(m.explorerView.Prefixes)
			if m.explorerView.SelectedObject >= prefixLen {
				objIdx := m.explorerView.SelectedObject - prefixLen
				if objIdx < len(m.explorerView.Objects) {
					targetKey := m.explorerView.Objects[objIdx].Key
					m.presignModal.Active = true
					m.presignModal.ObjectKey = targetKey
					m.presignModal.GeneratedURL = ""
					m.presignModal.ErrorText = ""
					m.presignModal.Input.SetValue("60m")
					m.presignModal.Input.Focus()
				}
			} else {
				m.notification = "Presigned URL is only applicable to objects"
			}
		}
	}

	return m, cmds
}

func (m Model) View() string {
	topBar := m.tabBar.Render(m.width, m.activeAccount, m.activeRegion, m.activeBucket)

	availableHeight := m.height - 4
	if availableHeight < 10 {
		availableHeight = 10
	}

	var body string
	switch m.activeTab {
	case 0, 1:
		body = m.explorerView.Render(m.width, availableHeight)
	case 2:
		body = m.transfersView.Render(m.width, availableHeight)
	case 3:
		body = m.syncView.Render(m.width, availableHeight)
	}

	filterBar := ""
	if m.searchBar.Active || m.searchBar.Input.Value() != "" {
		filterBar = m.searchBar.Render(m.width)
	}

	helpKeys := []string{"[Tab] Switch Pane", "[j/k] Navigate", "[/] Filter", "[u] Upload", "[d] Download", "[?] Help"}
	bottomBar := m.statusBar.Render(m.width, helpKeys, 0, 0, m.notification)

	parts := []string{topBar}
	if filterBar != "" {
		parts = append(parts, filterBar)
	}
	parts = append(parts, body, bottomBar)

	mainView := lipgloss.JoinVertical(lipgloss.Left, parts...)

	if m.uploadModal.Active {
		return m.uploadModal.Render(m.width, m.height)
	}

	if m.downloadModal.Active {
		return m.downloadModal.Render(m.width, m.height)
	}

	if m.presignModal.Active {
		return m.presignModal.Render(m.width, m.height)
	}

	if m.showHelpModal {
		helpModal := components.NewModalDialog("HELP OVERLAY", m.helpView.Render(m.width-10, m.height-10), []string{"Close (Esc)"}, m.styles)
		return helpModal.Render(m.width, m.height)
	}

	if m.showDeleteModal {
		deleteModal := components.NewModalDialog(
			"CONFIRM DELETION",
			fmt.Sprintf("Are you sure you want to permanently delete object:\n%s", m.deleteTargetKey),
			[]string{"Yes (Enter)", "Cancel (Esc)"},
			m.styles,
		)
		return deleteModal.Render(m.width, m.height)
	}

	return mainView
}

func (m Model) loadAccountsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services.AccountService == nil {
			return messages.AccountsLoadedMsg{}
		}
		accs, err := m.services.AccountService.ListAccounts(context.Background())
		if err != nil {
			return messages.AccountsLoadedMsg{Err: err}
		}
		result := make([]accDomain.Account, 0, len(accs))
		for _, a := range accs {
			if a != nil {
				result = append(result, *a)
			}
		}
		return messages.AccountsLoadedMsg{Accounts: result}
	}
}
func (m Model) loadBucketsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services.BucketService == nil {
			return messages.BucketsLoadedMsg{
				Buckets: []bucketDomain.Bucket{},
			}
		}
		b, err := m.services.BucketService.ListBuckets(context.Background())
		return messages.BucketsLoadedMsg{Buckets: b, Err: err}
	}
}

func (m Model) loadObjectsCmd(bucket, prefix string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.ObjectsLoadedMsg{
				Result: objPorts.ListObjectsResult{
					Objects:        []objDomain.Object{},
					CommonPrefixes: []objDomain.Prefix{},
				},
			}
		}
		res, err := m.services.ObjectService.BrowsePrefix(context.Background(), bucket, prefix, "")
		return messages.ObjectsLoadedMsg{Result: res, Err: err}
	}
}

func (m Model) loadObjectMetadataCmd(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.ObjectMetadataLoadedMsg{
				Metadata: objDomain.ObjectMetadata{},
			}
		}
		meta, err := m.services.ObjectService.GetObjectMetadata(context.Background(), bucket, key, "")
		tags, _ := m.services.ObjectService.GetObjectTags(context.Background(), bucket, key, "")
		return messages.ObjectMetadataLoadedMsg{Metadata: meta, Tags: tags, Err: err}
	}
}

func (m Model) inspectCurrentObjectCmd() tea.Cmd {
	prefixLen := len(m.explorerView.Prefixes)
	if m.explorerView.SelectedObject >= prefixLen {
		objIdx := m.explorerView.SelectedObject - prefixLen
		if objIdx < len(m.explorerView.Objects) {
			key := m.explorerView.Objects[objIdx].Key
			return tea.Batch(
				m.loadObjectMetadataCmd(m.activeBucket, key),
				m.loadObjectContentCmd(m.activeBucket, key),
			)
		}
	}
	return nil
}

func (m Model) loadObjectContentCmd(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.ContentPreviewLoadedMsg{
				Key: key,
				Err: nil,
			}
		}
		obj, err := m.services.ObjectService.GetObject(context.Background(), bucket, key, "")
		if err != nil {
			return messages.ContentPreviewLoadedMsg{Key: key, Err: err}
		}
		defer obj.Body.Close()

		buf := make([]byte, 4096)
		n, _ := io.ReadFull(obj.Body, buf)
		return messages.ContentPreviewLoadedMsg{
			Key:     key,
			Content: string(buf[:n]),
		}
	}
}

func (m Model) uploadObjectCmd(bucket, key, localPath string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.UploadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    fmt.Errorf("object service unavailable"),
			}
		}

		f, err := os.Open(localPath)
		if err != nil {
			return messages.UploadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    err,
			}
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return messages.UploadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    err,
			}
		}

		fullKey := key
		if m.explorerView.CurrentPrefix != "" && !strings.HasPrefix(fullKey, m.explorerView.CurrentPrefix) {
			fullKey = filepath.ToSlash(filepath.Join(m.explorerView.CurrentPrefix, key))
		}

		meta := objDomain.ObjectMetadata{
			ContentType:   "application/octet-stream",
			ContentLength: stat.Size(),
		}

		_, err = m.services.ObjectService.PutObject(context.Background(), bucket, fullKey, f, stat.Size(), meta)
		return messages.UploadFinishedMsg{
			Bucket: bucket,
			Key:    fullKey,
			Err:    err,
		}
	}
}

func (m Model) loadTransfersCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services.TransferService == nil {
			return messages.TransfersLoadedMsg{
				Jobs: []transferDomain.TransferJob{},
			}
		}
		jobs, err := m.services.TransferService.ListJobs(context.Background(), m.activeAccount, "")
		if err != nil || len(jobs) == 0 {
			return messages.TransfersLoadedMsg{
				Jobs: []transferDomain.TransferJob{},
			}
		}
		return messages.TransfersLoadedMsg{
			Jobs: jobs,
		}
	}
}

func (m Model) deleteObjectCmd(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService != nil {
			_ = m.services.ObjectService.DeleteObject(context.Background(), bucket, key, "")
		}
		return messages.StatusNotificationMsg{Message: fmt.Sprintf("Deleted: %s", key)}
	}
}

func (m Model) filterObjectsCmd(pattern string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService != nil {
			found, err := m.services.ObjectService.SearchObjects(context.Background(), m.activeBucket, m.explorerView.CurrentPrefix, pattern)
			return messages.ObjectsLoadedMsg{
				Result: objPorts.ListObjectsResult{
					Objects: found,
				},
				Err: err,
			}
		}
		return nil
	}
}

func (m Model) uploadFolderCmd(bucket, prefix, localDirPath string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.FolderUploadFinishedMsg{
				Bucket: bucket,
				Prefix: prefix,
				Err:    fmt.Errorf("object service unavailable"),
			}
		}

		cleanDir := filepath.Clean(localDirPath)
		var totalCount int
		var totalBytes int64
		var failedCount int

		err := filepath.Walk(cleanDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				failedCount++
				return nil
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(cleanDir, path)
			if err != nil {
				failedCount++
				return nil
			}

			targetKey := filepath.ToSlash(relPath)
			if prefix != "" {
				targetKey = filepath.ToSlash(filepath.Join(prefix, targetKey))
			}

			f, err := os.Open(path)
			if err != nil {
				failedCount++
				return nil
			}
			defer f.Close()

			meta := objDomain.ObjectMetadata{
				ContentType:   "application/octet-stream",
				ContentLength: info.Size(),
			}

			if m.services.TransferService != nil {
				job := transferDomain.TransferJob{
					ID:              fmt.Sprintf("up-%d", time.Now().UnixNano()),
					AccountID:       m.activeAccount,
					Type:            transferDomain.TransferTypeUpload,
					SourcePath:      path,
					DestinationPath: fmt.Sprintf("s3://%s/%s", bucket, targetKey),
					Bucket:          bucket,
					Key:             targetKey,
					TotalBytes:      info.Size(),
					Status:          transferDomain.JobStatusPending,
				}
				_, _ = m.services.TransferService.SubmitJob(context.Background(), job)
			}

			_, putErr := m.services.ObjectService.PutObject(context.Background(), bucket, targetKey, f, info.Size(), meta)
			if putErr != nil {
				failedCount++
				return nil
			}

			totalCount++
			totalBytes += info.Size()
			return nil
		})

		return messages.FolderUploadFinishedMsg{
			Bucket:      bucket,
			Prefix:      prefix,
			TotalCount:  totalCount,
			TotalBytes:  totalBytes,
			FailedCount: failedCount,
			Err:         err,
		}
	}
}

func (m Model) downloadObjectCmd(bucket, key, localDestDir string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    fmt.Errorf("object service unavailable"),
			}
		}

		if err := os.MkdirAll(localDestDir, 0755); err != nil {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    err,
			}
		}

		fileName := filepath.Base(key)
		destPath := filepath.Join(localDestDir, fileName)

		content, err := m.services.ObjectService.GetObject(context.Background(), bucket, key, "")
		if err != nil {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    err,
			}
		}
		defer content.Body.Close()

		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    err,
			}
		}
		defer outFile.Close()

		n, err := io.Copy(outFile, content.Body)
		if err != nil {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    err,
			}
		}

		if m.services.TransferService != nil {
			job := transferDomain.TransferJob{
				ID:               fmt.Sprintf("dl-%d", time.Now().UnixNano()),
				AccountID:        m.activeAccount,
				Type:             transferDomain.TransferTypeDownload,
				SourcePath:       fmt.Sprintf("s3://%s/%s", bucket, key),
				DestinationPath:  destPath,
				Bucket:           bucket,
				Key:              key,
				TotalBytes:       n,
				BytesTransferred: n,
				Status:           transferDomain.JobStatusCompleted,
			}
			_, _ = m.services.TransferService.SubmitJob(context.Background(), job)
		}

		return messages.DownloadFinishedMsg{
			Bucket: bucket,
			Key:    key,
			Path:   destPath,
			Err:    nil,
		}
	}
}

func (m Model) downloadFolderCmd(bucket, prefix, localDestDir string) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.FolderDownloadFinishedMsg{
				Bucket:    bucket,
				Prefix:    prefix,
				LocalDest: localDestDir,
				Err:       fmt.Errorf("object service unavailable"),
			}
		}

		var totalCount int
		var totalBytes int64
		var failedCount int

		var continuationToken string
		for {
			res, err := m.services.ObjectService.ListObjects(context.Background(), bucket, objDomain.ObjectFilter{
				Prefix:       prefix,
				MaxKeys:      1000,
				Continuation: continuationToken,
			})
			if err != nil {
				return messages.FolderDownloadFinishedMsg{
					Bucket:      bucket,
					Prefix:      prefix,
					LocalDest:   localDestDir,
					TotalCount:  totalCount,
					TotalBytes:  totalBytes,
					FailedCount: failedCount,
					Err:         err,
				}
			}

			for _, obj := range res.Objects {
				relKey := strings.TrimPrefix(obj.Key, prefix)
				relKey = strings.TrimPrefix(relKey, "/")
				if relKey == "" {
					continue
				}

				targetFilePath := filepath.Join(localDestDir, filepath.FromSlash(relKey))
				if err := os.MkdirAll(filepath.Dir(targetFilePath), 0755); err != nil {
					failedCount++
					continue
				}

				content, err := m.services.ObjectService.GetObject(context.Background(), bucket, obj.Key, "")
				if err != nil {
					failedCount++
					continue
				}

				outFile, err := os.OpenFile(targetFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					content.Body.Close()
					failedCount++
					continue
				}

				written, copyErr := io.Copy(outFile, content.Body)
				content.Body.Close()
				outFile.Close()

				if copyErr != nil {
					failedCount++
					continue
				}

				totalCount++
				totalBytes += written

				if m.services.TransferService != nil {
					job := transferDomain.TransferJob{
						ID:               fmt.Sprintf("dl-%d", time.Now().UnixNano()),
						AccountID:        m.activeAccount,
						Type:             transferDomain.TransferTypeDownload,
						SourcePath:       fmt.Sprintf("s3://%s/%s", bucket, obj.Key),
						DestinationPath:  targetFilePath,
						Bucket:           bucket,
						Key:              obj.Key,
						TotalBytes:       written,
						BytesTransferred: written,
						Status:           transferDomain.JobStatusCompleted,
					}
					_, _ = m.services.TransferService.SubmitJob(context.Background(), job)
				}
			}

			if !res.IsTruncated || res.NextContinuationToken == "" {
				break
			}
			continuationToken = res.NextContinuationToken
		}

		return messages.FolderDownloadFinishedMsg{
			Bucket:      bucket,
			Prefix:      prefix,
			LocalDest:   localDestDir,
			TotalCount:  totalCount,
			TotalBytes:  totalBytes,
			FailedCount: failedCount,
			Err:         nil,
		}
	}
}

func (m Model) generatePresignedURLCmd(bucket, key string, expiry time.Duration) tea.Cmd {
	return func() tea.Msg {
		if m.services.ObjectService == nil {
			return messages.PresignedURLGeneratedMsg{
				Key: key,
				Err: fmt.Errorf("object service unavailable"),
			}
		}

		pURL, err := m.services.ObjectService.GeneratePresignedURL(context.Background(), bucket, key, "GET", expiry)
		if err != nil {
			return messages.PresignedURLGeneratedMsg{
				Key: key,
				Err: err,
			}
		}

		seq := osc52.New(pURL.URL)
		fmt.Fprint(os.Stderr, seq)

		return messages.PresignedURLGeneratedMsg{
			URL:    pURL.URL,
			Key:    key,
			Copied: true,
			Err:    nil,
		}
	}
}

func (m Model) loadSyncJobsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services.SyncRepo == nil {
			return messages.SyncJobsLoadedMsg{
				Jobs: []*syncPorts.SyncJobRecord{},
			}
		}
		jobs, err := m.services.SyncRepo.ListJobs(context.Background(), m.activeAccount)
		return messages.SyncJobsLoadedMsg{
			Jobs: jobs,
			Err:  err,
		}
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
