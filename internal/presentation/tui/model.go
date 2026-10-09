package tui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	"github.com/sekai-labs/kumokura/internal/presentation/tui/components/filepicker"
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

	explorerView    views.ExplorerView
	transfersView   views.TransfersView
	syncView        views.SyncView
	helpView        views.HelpView
	activeAccount   string
	activeAccountID string
	activeRegion    string
	activeBucket    string
	showHelpModal   bool
	showDeleteModal bool
	deleteTargetKey string
	uploadModal     components.UploadModal
	yaziPicker      filepicker.YaziPicker
	downloadModal   components.DownloadModal
	presignModal    components.PresignModal
	activeUploads   int
	notification string
	notifTimerID int
}

type clearNotificationMsg struct {
	id int
}

type transferTickMsg time.Time

func transferTickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return transferTickMsg(t)
	})
}

type progressReader struct {
	reader      io.Reader
	total       int64
	transferred int64
	onProgress  func(int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.transferred += int64(n)
		if pr.onProgress != nil {
			pr.onProgress(pr.transferred)
		}
	}
	return n, err
}

func NewModel(services Services) Model {
	st := styles.NewStyles(styles.DarkTheme)
	tabs := []components.TabItem{
		{ID: "buckets", Title: "1 Buckets"},
		{ID: "objects", Title: "2 Objects"},
		{ID: "transfers", Title: "3 Transfers"},
		{ID: "sync", Title: "4 Sync"},
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
		yaziPicker:    filepicker.NewYaziPicker(".", "", "", st),
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
		transferTickCmd(),
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
			m.activeAccountID = string(msg.Accounts[0].ID)
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
		if msg.Err != nil {
			m.notification = fmt.Sprintf("Error loading objects: %v", msg.Err)
		} else {
			m.explorerView.Objects = msg.Result.Objects
			m.explorerView.Prefixes = msg.Result.CommonPrefixes
			m.explorerView.SelectedObject = 0
			m.explorerView.ObjectOffset = 0
			if len(msg.Result.Objects) > 0 {
				cmds = append(cmds, m.loadObjectMetadataCmd(m.activeBucket, msg.Result.Objects[0].Key))
			} else {
				m.explorerView.PreviewMetadata = nil
				m.explorerView.PreviewContent = nil
				m.explorerView.PreviewTags = nil
			}
		}
	case messages.ObjectMetadataLoadedMsg:
		if msg.Err == nil {
			if msg.Metadata.ContentType == "" && msg.Metadata.ContentLength == 0 {
				m.explorerView.PreviewMetadata = nil
				m.explorerView.PreviewTags = nil
				m.explorerView.PreviewContent = nil
			} else {
				m.explorerView.PreviewMetadata = &msg.Metadata
				m.explorerView.PreviewTags = msg.Tags
			}
		}

	case messages.ContentPreviewLoadedMsg:
		if msg.Err == nil {
			if msg.Key == "" {
				m.explorerView.PreviewContent = nil
			} else {
				m.explorerView.PreviewContent = []byte(msg.Content)
			}
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
	case messages.BatchUploadFinishedMsg:
		m.activeUploads--
		if m.activeUploads < 0 {
			m.activeUploads = 0
		}
		if msg.Err != nil {
			m.notification = fmt.Sprintf("Batch upload encountered errors: %v", msg.Err)
		} else {
			m.notification = fmt.Sprintf("Finished uploading %d items (%s)", msg.TotalCount, formatBytes(msg.TotalBytes))
		}
		cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, m.explorerView.CurrentPrefix))
		cmds = append(cmds, m.loadTransfersCmd())

	case transferTickMsg:
		cmds = append(cmds, transferTickCmd())
		cmds = append(cmds, m.loadTransfersCmd())

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
		m.notifTimerID++
		currentID := m.notifTimerID
		cmds = append(cmds, tea.Tick(4*time.Second, func(time.Time) tea.Msg {
			return clearNotificationMsg{id: currentID}
		}))

	case clearNotificationMsg:
		if msg.id == m.notifTimerID {
			m.notification = ""
		}
	case tea.MouseMsg:
		return m, nil
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

		if m.yaziPicker.Active {
			switch msg.String() {
			case "esc", "q":
				m.yaziPicker.Active = false
				m.uploadModal.Active = false
				return m, nil
			case "k", "up":
				m.yaziPicker.MoveUp()
				return m, nil
			case "j", "down":
				m.yaziPicker.MoveDown()
				return m, nil
			case "h", "left", "backspace":
				m.yaziPicker.ParentDirectory()
				return m, nil
			case "l", "right":
				m.yaziPicker.EnterDir()
				return m, nil
			case " ":
				m.yaziPicker.ToggleSelect()
				return m, nil
			case "a":
				m.yaziPicker.SelectAll()
				return m, nil
			case "tab":
				m.yaziPicker.ToggleDirMode()
				return m, nil
			case "enter", "u", "y":
				selectedPaths := m.yaziPicker.GetSelectedPaths()
				m.yaziPicker.Active = false
				if len(selectedPaths) == 0 {
					m.notification = "No files or directories selected"
					return m, nil
				}
				m.activeUploads++
				m.notification = fmt.Sprintf("Uploading %d item(s) to s3://%s/%s...", len(selectedPaths), m.activeBucket, m.explorerView.CurrentPrefix)
				cmds = append(cmds, m.uploadBatchCmd(m.activeBucket, m.explorerView.CurrentPrefix, selectedPaths))
				cmds = append(cmds, m.loadTransfersCmd())
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

		if m.uploadModal.Active {
			switch {
			case key.Matches(msg, m.keymap.Escape):
				m.uploadModal.Active = false
				m.uploadModal.Input.Blur()
			case key.Matches(msg, m.keymap.Enter):
				rawPath := strings.TrimSpace(m.uploadModal.Input.Value())
				path := cleanLocalInputPath(rawPath)
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

				target := strings.TrimSpace(m.downloadModal.TargetName)
				if target == "" {
					m.notification = "No download target specified"
					return m, nil
				}

				if m.downloadModal.IsFolder {
					m.notification = fmt.Sprintf("Downloading folder %s to %s...", target, destDir)
					cmds = append(cmds, m.downloadFolderCmd(m.activeBucket, target, destDir))
				} else {
					m.notification = fmt.Sprintf("Downloading %s to %s...", filepath.Base(target), destDir)
					cmds = append(cmds, m.downloadObjectCmd(m.activeBucket, target, destDir))
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
			if m.activeTab == 0 || m.activeTab == 1 {
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

	case key.Matches(msg, m.keymap.Enter), key.Matches(msg, m.keymap.Right):
		if m.explorerView.ActivePaneIndex == 0 {
			if len(m.explorerView.Buckets) > m.explorerView.SelectedBucket {
				m.activeBucket = m.explorerView.Buckets[m.explorerView.SelectedBucket].Name
				m.explorerView.ActiveBucket = m.activeBucket
				m.explorerView.CurrentPrefix = ""
				m.explorerView.SelectedObject = 0
				m.explorerView.ObjectOffset = 0
				m.explorerView.PreviewMetadata = nil
				m.explorerView.PreviewContent = nil
				m.explorerView.PreviewTags = nil
				m.explorerView.ActivePaneIndex = 1
				cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, ""))
			}
		} else if m.explorerView.ActivePaneIndex == 1 {
			if m.explorerView.SelectedObject < len(m.explorerView.Prefixes) {
				targetPrefix := m.explorerView.Prefixes[m.explorerView.SelectedObject].Prefix
				m.explorerView.CurrentPrefix = targetPrefix
				m.explorerView.SelectedObject = 0
				m.explorerView.ObjectOffset = 0
				m.explorerView.PreviewMetadata = nil
				m.explorerView.PreviewContent = nil
				m.explorerView.PreviewTags = nil
				cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, targetPrefix))
			} else {
				objIdx := m.explorerView.SelectedObject - len(m.explorerView.Prefixes)
				if objIdx < len(m.explorerView.Objects) {
					obj := m.explorerView.Objects[objIdx]
					if strings.HasSuffix(obj.Key, "/") {
						m.explorerView.CurrentPrefix = obj.Key
						m.explorerView.SelectedObject = 0
						m.explorerView.ObjectOffset = 0
						m.explorerView.PreviewMetadata = nil
						m.explorerView.PreviewContent = nil
						m.explorerView.PreviewTags = nil
						cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, obj.Key))
					} else {
						m.explorerView.ShowPreview = true
						cmds = append(cmds, m.inspectCurrentObjectCmd())
					}
				}
			}
		}

	case key.Matches(msg, m.keymap.Back), key.Matches(msg, m.keymap.Left):
		if m.explorerView.CurrentPrefix != "" {
			trimmed := strings.TrimSuffix(m.explorerView.CurrentPrefix, "/")
			lastSlash := strings.LastIndex(trimmed, "/")
			if lastSlash >= 0 {
				m.explorerView.CurrentPrefix = trimmed[:lastSlash+1]
			} else {
				m.explorerView.CurrentPrefix = ""
			}
			m.explorerView.SelectedObject = 0
			m.explorerView.ObjectOffset = 0
			m.explorerView.PreviewMetadata = nil
			m.explorerView.PreviewContent = nil
			m.explorerView.PreviewTags = nil
			cmds = append(cmds, m.loadObjectsCmd(m.activeBucket, m.explorerView.CurrentPrefix))
		} else if m.explorerView.ActivePaneIndex == 1 {
			m.explorerView.ActivePaneIndex = 0
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
			cwd, _ := os.Getwd()
			m.yaziPicker = filepicker.NewYaziPicker(cwd, m.activeBucket, m.explorerView.CurrentPrefix, m.styles)
			m.yaziPicker.Active = true
			m.uploadModal.Active = true
			m.uploadModal.Destination = fmt.Sprintf("s3://%s/%s", m.activeBucket, m.explorerView.CurrentPrefix)
			m.uploadModal.ErrorText = ""
			m.uploadModal.Input.SetValue("")
		}

	case key.Matches(msg, m.keymap.Download):
		if m.activeBucket == "" {
			m.notification = "Select a bucket before downloading"
		} else {
			prefixLen := len(m.explorerView.Prefixes)
			objLen := len(m.explorerView.Objects)
			totalLen := prefixLen + objLen
			if totalLen == 0 {
				m.notification = "No object or folder selected"
			} else {
				if m.explorerView.SelectedObject < 0 {
					m.explorerView.SelectedObject = 0
				}
				if m.explorerView.SelectedObject >= totalLen {
					m.explorerView.SelectedObject = totalLen - 1
				}
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
					if objIdx >= 0 && objIdx < objLen {
						targetKey := m.explorerView.Objects[objIdx].Key
						m.downloadModal.Active = true
						m.downloadModal.TargetName = targetKey
						m.downloadModal.IsFolder = false
						m.downloadModal.ErrorText = ""
						m.downloadModal.Input.SetValue("./")
						m.downloadModal.Input.Focus()
					} else {
						m.notification = "No object or folder selected"
					}
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

	var helpKeys []string
	if m.width >= 120 {
		helpKeys = []string{"[Tab] Switch Pane", "[j/k] Navigate", "[/] Filter", "[u] Upload", "[d] Download", "[p] Presigned", "[i] Inspect", "[?] Help"}
	} else {
		helpKeys = []string{"[Tab] Switch Pane", "[j/k] Navigate", "[/] Filter", "[u] Upload", "[d] Download", "[?] Help"}
	}
	bottomBar := m.statusBar.Render(m.width, helpKeys, 0, 0, m.notification)

	parts := []string{topBar}
	if filterBar != "" {
		parts = append(parts, filterBar)
	}
	parts = append(parts, body, bottomBar)

	mainView := lipgloss.JoinVertical(lipgloss.Left, parts...)

	if m.yaziPicker.Active {
		return m.yaziPicker.Render(m.width, m.height)
	}

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

func (m *Model) clearPreview() {
	m.explorerView.PreviewMetadata = nil
	m.explorerView.PreviewContent = nil
	m.explorerView.PreviewTags = nil
}

func (m Model) inspectCurrentObjectCmd() tea.Cmd {
	prefixLen := len(m.explorerView.Prefixes)
	if m.explorerView.SelectedObject < prefixLen {
		return func() tea.Msg {
			return messages.ObjectMetadataLoadedMsg{}
		}
	}
	objIdx := m.explorerView.SelectedObject - prefixLen
	if objIdx < len(m.explorerView.Objects) {
		key := m.explorerView.Objects[objIdx].Key
		if strings.HasSuffix(key, "/") {
			return func() tea.Msg {
				return messages.ObjectMetadataLoadedMsg{}
			}
		}
		return tea.Batch(
			m.loadObjectMetadataCmd(m.activeBucket, key),
			m.loadObjectContentCmd(m.activeBucket, key),
		)
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

		accountID := m.resolveAccountID()
		jobID := fmt.Sprintf("up-%d", time.Now().UnixNano())
		if m.services.TransferService != nil {
			job := transferDomain.TransferJob{
				ID:               jobID,
				AccountID:        accountID,
				Type:             transferDomain.TransferTypeUpload,
				SourcePath:       localPath,
				DestinationPath:  fmt.Sprintf("s3://%s/%s", bucket, fullKey),
				Bucket:           bucket,
				Key:              fullKey,
				TotalBytes:       stat.Size(),
				BytesTransferred: 0,
				Status:           transferDomain.JobStatusRunning,
			}
			_, _ = m.services.TransferService.SubmitJob(context.Background(), job)
		}

		meta := objDomain.ObjectMetadata{
			ContentType:   "application/octet-stream",
			ContentLength: stat.Size(),
		}

		pr := &progressReader{
			reader: f,
			total:  stat.Size(),
			onProgress: func(transferred int64) {
				if m.services.TransferService != nil {
					_ = m.services.TransferService.UpdateJobProgress(context.Background(), jobID, transferred)
				}
			},
		}

		_, err = m.services.ObjectService.PutObject(context.Background(), bucket, fullKey, pr, stat.Size(), meta)
		if m.services.TransferService != nil {
			if err != nil {
				_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobID, transferDomain.JobStatusFailed, err.Error())
			} else {
				_ = m.services.TransferService.UpdateJobProgress(context.Background(), jobID, stat.Size())
				_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobID, transferDomain.JobStatusCompleted, "")
			}
		}

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
		accountID := m.resolveAccountID()
		jobs, err := m.services.TransferService.ListJobs(context.Background(), accountID, "")
		if (err != nil || len(jobs) == 0) && accountID != m.activeAccount {
			jobs, err = m.services.TransferService.ListJobs(context.Background(), m.activeAccount, "")
		}
		if (err != nil || len(jobs) == 0) && accountID != "" {
			jobs, err = m.services.TransferService.ListJobs(context.Background(), "", "")
		}
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
		folderBase := filepath.Base(cleanDir)
		targetFolderPrefix := filepath.ToSlash(filepath.Clean(prefix))
		if targetFolderPrefix == "." || targetFolderPrefix == "/" {
			targetFolderPrefix = ""
		}
		if targetFolderPrefix != "" && !strings.HasSuffix(targetFolderPrefix, "/") {
			targetFolderPrefix += "/"
		}
		folderS3Prefix := targetFolderPrefix + folderBase + "/"

		accountID := m.resolveAccountID()

		rootMarkerMeta := objDomain.ObjectMetadata{
			ContentType:   "application/x-directory",
			ContentLength: 0,
		}
		jobIDRoot := fmt.Sprintf("up-%d", time.Now().UnixNano())
		if m.services.TransferService != nil {
			job := transferDomain.TransferJob{
				ID:               jobIDRoot,
				AccountID:        accountID,
				Type:             transferDomain.TransferTypeUpload,
				SourcePath:       cleanDir,
				DestinationPath:  fmt.Sprintf("s3://%s/%s", bucket, folderS3Prefix),
				Bucket:           bucket,
				Key:              folderS3Prefix,
				TotalBytes:       0,
				BytesTransferred: 0,
				Status:           transferDomain.JobStatusRunning,
			}
			_, _ = m.services.TransferService.SubmitJob(context.Background(), job)
		}
		_, rootMarkerErr := m.services.ObjectService.PutObject(context.Background(), bucket, folderS3Prefix, bytes.NewReader([]byte{}), 0, rootMarkerMeta)
		if m.services.TransferService != nil {
			if rootMarkerErr != nil {
				_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobIDRoot, transferDomain.JobStatusFailed, rootMarkerErr.Error())
			} else {
				_ = m.services.TransferService.UpdateJobProgress(context.Background(), jobIDRoot, 0)
				_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobIDRoot, transferDomain.JobStatusCompleted, "")
			}
		}

		type uploadTask struct {
			isDir    bool
			localPath string
			s3Key    string
			size     int64
		}

		var tasks []uploadTask
		var walkErr error

		walkErr = filepath.Walk(cleanDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			relPath, err := filepath.Rel(cleanDir, path)
			if err != nil || relPath == "." {
				return nil
			}

			if info.IsDir() {
				dirMarkerKey := folderS3Prefix + filepath.ToSlash(relPath) + "/"
				tasks = append(tasks, uploadTask{
					isDir:     true,
					localPath: path,
					s3Key:     dirMarkerKey,
					size:      0,
				})
			} else {
				targetKey := folderS3Prefix + filepath.ToSlash(relPath)
				tasks = append(tasks, uploadTask{
					isDir:     false,
					localPath: path,
					s3Key:     targetKey,
					size:      info.Size(),
				})
			}
			return nil
		})

		concurrency := 16
		taskCh := make(chan uploadTask, len(tasks))
		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)

		var totalCount int
		var totalBytes int64
		var failedCount int
		var mu sync.Mutex
		var wg sync.WaitGroup

		for range concurrency {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for t := range taskCh {
					if t.isDir {
						dirMeta := objDomain.ObjectMetadata{
							ContentType:   "application/x-directory",
							ContentLength: 0,
						}
						jobID := fmt.Sprintf("up-%d", time.Now().UnixNano())
						if m.services.TransferService != nil {
							j := transferDomain.TransferJob{
								ID:               jobID,
								AccountID:        accountID,
								Type:             transferDomain.TransferTypeUpload,
								SourcePath:       t.localPath,
								DestinationPath:  fmt.Sprintf("s3://%s/%s", bucket, t.s3Key),
								Bucket:           bucket,
								Key:              t.s3Key,
								Status:           transferDomain.JobStatusRunning,
							}
							_, _ = m.services.TransferService.SubmitJob(context.Background(), j)
						}
						_, pErr := m.services.ObjectService.PutObject(context.Background(), bucket, t.s3Key, bytes.NewReader([]byte{}), 0, dirMeta)
						if m.services.TransferService != nil {
							if pErr != nil {
								_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobID, transferDomain.JobStatusFailed, pErr.Error())
							} else {
								_ = m.services.TransferService.UpdateJobProgress(context.Background(), jobID, 0)
								_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobID, transferDomain.JobStatusCompleted, "")
							}
						}
						mu.Lock()
						if pErr != nil {
							failedCount++
						}
						mu.Unlock()
					} else {
						f, openErr := os.Open(t.localPath)
						if openErr != nil {
							mu.Lock()
							failedCount++
							mu.Unlock()
							continue
						}

						meta := objDomain.ObjectMetadata{
							ContentType:   "application/octet-stream",
							ContentLength: t.size,
						}
						jobID := fmt.Sprintf("up-%d", time.Now().UnixNano())
						if m.services.TransferService != nil {
							j := transferDomain.TransferJob{
								ID:               jobID,
								AccountID:        accountID,
								Type:             transferDomain.TransferTypeUpload,
								SourcePath:       t.localPath,
								DestinationPath:  fmt.Sprintf("s3://%s/%s", bucket, t.s3Key),
								Bucket:           bucket,
								Key:              t.s3Key,
								TotalBytes:       t.size,
								Status:           transferDomain.JobStatusRunning,
							}
							_, _ = m.services.TransferService.SubmitJob(context.Background(), j)
						}

						pr := &progressReader{
							reader: f,
							total:  t.size,
							onProgress: func(transferred int64) {
								if m.services.TransferService != nil {
									_ = m.services.TransferService.UpdateJobProgress(context.Background(), jobID, transferred)
								}
							},
						}

						_, putErr := m.services.ObjectService.PutObject(context.Background(), bucket, t.s3Key, pr, t.size, meta)
						f.Close()

						if m.services.TransferService != nil {
							if putErr != nil {
								_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobID, transferDomain.JobStatusFailed, putErr.Error())
							} else {
								_ = m.services.TransferService.UpdateJobProgress(context.Background(), jobID, t.size)
								_ = m.services.TransferService.UpdateJobStatus(context.Background(), jobID, transferDomain.JobStatusCompleted, "")
							}
						}

						mu.Lock()
						if putErr != nil {
							failedCount++
						} else {
							totalCount++
							totalBytes += t.size
						}
						mu.Unlock()
					}
				}
			}()
		}
		wg.Wait()

		return messages.FolderUploadFinishedMsg{
			Bucket:      bucket,
			Prefix:      folderS3Prefix,
			TotalCount:  totalCount,
			TotalBytes:  totalBytes,
			FailedCount: failedCount,
			Err:         walkErr,
		}
	}
}

func (m Model) uploadBatchCmd(bucket, prefix string, paths []string) tea.Cmd {
	return func() tea.Msg {
		var totalCount int
		var totalBytes int64
		var failedCount int

		for _, p := range paths {
			fi, err := os.Stat(p)
			if err != nil {
				failedCount++
				continue
			}
			if fi.IsDir() {
				folderMsg := m.uploadFolderCmd(bucket, prefix, p)()
				if fm, ok := folderMsg.(messages.FolderUploadFinishedMsg); ok {
					totalCount += fm.TotalCount
					totalBytes += fm.TotalBytes
					failedCount += fm.FailedCount
				}
			} else {
				fileName := filepath.Base(p)
				targetKey := fileName
				if prefix != "" {
					targetKey = filepath.ToSlash(filepath.Join(prefix, fileName))
				}
				objMsg := m.uploadObjectCmd(bucket, targetKey, p)()
				if om, ok := objMsg.(messages.UploadFinishedMsg); ok {
					if om.Err != nil {
						failedCount++
					} else {
						totalCount++
						totalBytes += fi.Size()
					}
				}
			}
		}

		var finalErr error
		if failedCount > 0 && totalCount == 0 {
			finalErr = fmt.Errorf("%d item(s) failed to upload", failedCount)
		}

		return messages.BatchUploadFinishedMsg{
			Bucket:      bucket,
			Prefix:      prefix,
			TotalCount:  totalCount,
			TotalBytes:  totalBytes,
			FailedCount: failedCount,
			Err:         finalErr,
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

		key = strings.TrimSpace(key)
		if key == "" {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    fmt.Errorf("empty object key"),
			}
		}

		if strings.TrimSpace(localDestDir) == "" {
			localDestDir = "."
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
		if content.Body == nil {
			return messages.DownloadFinishedMsg{
				Bucket: bucket,
				Key:    key,
				Err:    fmt.Errorf("empty response body from storage"),
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
			accountID := m.resolveAccountID()
			job := transferDomain.TransferJob{
				ID:               fmt.Sprintf("dl-%d", time.Now().UnixNano()),
				AccountID:        accountID,
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
				cleanRelKey := filepath.Clean(filepath.FromSlash(relKey))
				if cleanRelKey == "." || cleanRelKey == ".." || strings.HasPrefix(cleanRelKey, ".."+string(filepath.Separator)) {
					failedCount++
					continue
				}

				targetFilePath := filepath.Join(localDestDir, cleanRelKey)
				if err := os.MkdirAll(filepath.Dir(targetFilePath), 0755); err != nil {
					failedCount++
					continue
				}
				content, err := m.services.ObjectService.GetObject(context.Background(), bucket, obj.Key, "")
				if err != nil {
					failedCount++
					continue
				}
				if content.Body == nil {
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
					accountID := m.resolveAccountID()
					job := transferDomain.TransferJob{
						ID:               fmt.Sprintf("dl-%d", time.Now().UnixNano()),
						AccountID:        accountID,
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
		accountID := m.resolveAccountID()
		jobs, err := m.services.SyncRepo.ListJobs(context.Background(), accountID)
		if (err != nil || len(jobs) == 0) && accountID != m.activeAccount {
			jobs, err = m.services.SyncRepo.ListJobs(context.Background(), m.activeAccount)
		}
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
func cleanLocalInputPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") && len(raw) >= 2 {
		raw = strings.Trim(raw, "\"")
	} else if strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") && len(raw) >= 2 {
		raw = strings.Trim(raw, "'")
	}
	raw = strings.TrimSpace(raw)
	if raw == "~" || strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			if raw == "~" {
				raw = home
			} else {
				raw = filepath.Join(home, raw[2:])
			}
		}
	}
	return filepath.Clean(raw)
}

func (m Model) resolveAccountID() string {
	if m.activeAccountID != "" {
		return m.activeAccountID
	}
	if m.services.AccountService != nil {
		if accs, err := m.services.AccountService.ListAccounts(context.Background()); err == nil {
			for _, a := range accs {
				if a != nil && (a.Name == m.activeAccount || string(a.ID) == m.activeAccount) {
					return string(a.ID)
				}
			}
		}
	}
	return m.activeAccount
}
