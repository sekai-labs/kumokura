package desktop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	accountDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	accountPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/sekai-labs/kumokura/internal/bootstrap"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objectDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/platform/config"
	transferDomain "github.com/sekai-labs/kumokura/internal/transfers/domain"
)

type DesktopApp struct {
	container *bootstrap.AppContainer
	fyneApp   fyne.App
	window    fyne.Window

	mu              sync.RWMutex
	accounts        []*accountDomain.Account
	selectedAccount *accountDomain.Account

	buckets        []bucketDomain.Bucket
	selectedBucket string
	bucketFilter   string

	prefixes        []objectDomain.Prefix
	objects         []objectDomain.Object
	filteredObjects []objectDomain.Object
	selectedObject  *objectDomain.Object

	currentPrefix    string
	flatMode         bool
	searchQuery      string
	lastSelectedRow  int
	lastSelectedCol  int
	lastSelectedTime time.Time
	jobs             []transferDomain.TransferJob

	accountSelect *widget.Select
	bucketBadge   *widget.Label
	searchEntry   *widget.Entry
	endpointBadge *widget.Label

	accountList *widget.List
	bucketList  *widget.List

	bucketTabBadge *widget.Label
	bucketSearch   *widget.Entry

	objectTable         *widget.Table
	emptyStateCard      *widget.Card
	emptyStateMsg       *widget.Label
	emptyUploadBtn      *widget.Button
	emptyUploadDirBtn   *widget.Button
	emptyClearFilterBtn *widget.Button
	loadingContainer    *fyne.Container
	loadingBar          *widget.ProgressBarInfinite
	loadingLabel        *widget.Label
	centerContainer     *fyne.Container

	prefixNavLabel *widget.Label
	upFolderBtn    *widget.Button
	viewModeSelect *widget.Select
	openFolderBtn  *widget.Button
	loadCancel     context.CancelFunc
	loadSeq        uint64
	inspectCancel  context.CancelFunc
	detailsCard    *widget.Card
	metadataLabel  *widget.Label
	presignedLabel *widget.Entry
	tagsLabel      *widget.Label
	previewBtn     *widget.Button
	openExternalBtn *widget.Button

	previewContentEntry *widget.Entry
	previewImage        *canvas.Image
	previewImageWrap    *fyne.Container
	previewVideoBadge   *widget.Label
	previewVideoBox     *fyne.Container
	previewStatusLabel  *widget.Label
	inspectorSplit *container.Split
	inspectorOpen  bool

	transferList   *widget.List
	transfersTray  *fyne.Container
	transfersOpen  bool
	transferStatus *widget.Label

	breadcrumbContainer *fyne.Container
	toastLabel          *widget.Label
	toastTimer          *time.Timer

	refreshTicker *time.Ticker
	stopTicker    chan struct{}
}

func NewDesktopApp(appContainer *bootstrap.AppContainer) *DesktopApp {
	return NewDesktopAppWithFyneApp(appContainer, app.NewWithID("com.sekai.kumokura"))
}

func NewDesktopAppWithFyneApp(appContainer *bootstrap.AppContainer, a fyne.App) *DesktopApp {
	if a != nil && a.Settings() != nil {
		a.Settings().SetTheme(NewBlackTheme())
	}
	w := a.NewWindow("Kumokura - Native Object Storage Explorer")
	w.Resize(fyne.NewSize(1280, 800))

	da := &DesktopApp{
		container:       appContainer,
		fyneApp:         a,
		window:          w,
		lastSelectedRow: -1,
		lastSelectedCol: -1,
		stopTicker:      make(chan struct{}),
		inspectorOpen:   true,
		transfersOpen:   true,
	}
	da.buildUI()
	return da
}

func (d *DesktopApp) Run() {
	d.loadAccounts()
	d.startBackgroundPoller()
	d.window.ShowAndRun()
	close(d.stopTicker)
}

func (d *DesktopApp) showToast(message string) {
	if d.fyneApp != nil {
		d.fyneApp.SendNotification(fyne.NewNotification("Kumokura", message))
	}
	fyne.Do(func() {
		if d.toastLabel != nil {
			d.toastLabel.SetText(message)
			d.toastLabel.Show()
			if d.toastTimer != nil {
				d.toastTimer.Stop()
			}
			d.toastTimer = time.AfterFunc(4*time.Second, func() {
				fyne.Do(func() {
					if d.toastLabel != nil {
						d.toastLabel.Hide()
					}
				})
			})
		}
	})
}

func (d *DesktopApp) buildUI() {
	header := d.buildHeader()
	masterDetail := d.buildMasterDetail()
	bottomPanel := d.buildBottomPanel()

	content := container.NewBorder(
		header,
		bottomPanel,
		nil,
		nil,
		masterDetail,
	)

	d.window.SetContent(content)
}

func (d *DesktopApp) buildHeader() fyne.CanvasObject {
	brandLabel := widget.NewLabelWithStyle("☁ Kumokura", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	d.accountSelect = widget.NewSelect([]string{}, func(selected string) {
		d.onAccountSelectedByName(selected)
	})
	d.accountSelect.PlaceHolder = "Select Profile"

	addAccountBtn := widget.NewButtonWithIcon("Add Account", theme.ContentAddIcon(), func() {
		d.showAddAccountDialog()
	})

	d.endpointBadge = widget.NewLabel("(No Region)")
	d.endpointBadge.TextStyle = fyne.TextStyle{Italic: true}

	leftHeader := container.NewHBox(
		brandLabel,
		widget.NewSeparator(),
		widget.NewIcon(theme.AccountIcon()),
		d.accountSelect,
		addAccountBtn,
		d.endpointBadge,
	)

	d.bucketBadge = widget.NewLabel("No Bucket Selected")
	d.bucketBadge.TextStyle = fyne.TextStyle{Bold: true}

	d.breadcrumbContainer = container.NewHBox()
	d.updateBreadcrumbs()

	centerHeader := container.NewHBox(
		widget.NewIcon(theme.StorageIcon()),
		d.bucketBadge,
		widget.NewSeparator(),
		d.breadcrumbContainer,
	)

	d.searchEntry = widget.NewEntry()
	d.searchEntry.SetPlaceHolder("Instant search objects...")
	d.searchEntry.SetIcon(theme.SearchIcon())
	d.searchEntry.OnChanged = func(query string) {
		d.onSearchChanged(query)
	}

	clearSearchBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		if d.searchEntry != nil {
			d.searchEntry.SetText("")
		}
	})
	clearSearchBtn.Importance = widget.LowImportance

	searchBox := container.NewBorder(
		nil,
		nil,
		nil,
		clearSearchBtn,
		d.searchEntry,
	)

	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
		d.showSettingsDialog()
	})

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		d.refreshAll()
	})

	rightHeader := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(240, 36), searchBox),
		settingsBtn,
		refreshBtn,
	)

	topBar := container.NewBorder(
		nil,
		nil,
		leftHeader,
		rightHeader,
		container.NewCenter(centerHeader),
	)

	d.toastLabel = widget.NewLabel("")
	d.toastLabel.TextStyle = fyne.TextStyle{Italic: true, Bold: true}
	d.toastLabel.Hide()

	toastBar := container.NewHBox(
		widget.NewIcon(theme.InfoIcon()),
		d.toastLabel,
	)

	return container.NewVBox(
		topBar,
		toastBar,
		widget.NewSeparator(),
	)
}

func (d *DesktopApp) updateBreadcrumbs() {
	if d.breadcrumbContainer == nil {
		return
	}

	d.mu.RLock()
	bucket := d.selectedBucket
	prefix := d.currentPrefix
	isFlat := d.flatMode
	d.mu.RUnlock()

	d.breadcrumbContainer.Objects = nil

	if bucket == "" {
		lbl := widget.NewLabel("s3://")
		lbl.TextStyle = fyne.TextStyle{Monospace: true}
		d.breadcrumbContainer.Add(lbl)
		d.breadcrumbContainer.Refresh()
		return
	}

	bucketBtn := widget.NewButton(fmt.Sprintf("s3://%s", bucket), func() {
		d.navigateToPrefix("")
	})
	bucketBtn.Importance = widget.LowImportance
	d.breadcrumbContainer.Add(bucketBtn)

	if isFlat {
		lbl := widget.NewLabel("/ [flat view]")
		lbl.TextStyle = fyne.TextStyle{Monospace: true, Italic: true}
		d.breadcrumbContainer.Add(lbl)
		d.breadcrumbContainer.Refresh()
		return
	}

	if prefix != "" {
		parts := strings.Split(strings.TrimSuffix(prefix, "/"), "/")
		accumulated := ""
		for _, part := range parts {
			if part == "" {
				continue
			}
			accumulated += part + "/"
			targetPath := accumulated
			sep := widget.NewLabel("/")
			sep.TextStyle = fyne.TextStyle{Monospace: true}
			d.breadcrumbContainer.Add(sep)

			partBtn := widget.NewButton(part, func() {
				d.navigateToPrefix(targetPath)
			})
			partBtn.Importance = widget.LowImportance
			d.breadcrumbContainer.Add(partBtn)
		}
	}

	d.breadcrumbContainer.Refresh()
}

func (d *DesktopApp) buildMasterDetail() fyne.CanvasObject {
	leftPanel := d.buildLeftPanel()
	centerPanel := d.buildCenterPanel()
	rightPanel := d.buildRightPanel()

	d.inspectorSplit = container.NewHSplit(centerPanel, rightPanel)
	d.inspectorSplit.SetOffset(0.70)

	outerSplit := container.NewHSplit(leftPanel, d.inspectorSplit)
	outerSplit.SetOffset(0.24)

	return outerSplit
}

func (d *DesktopApp) getFilteredBuckets() []bucketDomain.Bucket {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.bucketFilter == "" {
		return d.buckets
	}

	q := strings.ToLower(d.bucketFilter)
	var filtered []bucketDomain.Bucket
	for _, b := range d.buckets {
		if strings.Contains(strings.ToLower(b.Name), q) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

func (d *DesktopApp) buildLeftPanel() fyne.CanvasObject {
	accountHeader := container.NewHBox(
		widget.NewIcon(theme.AccountIcon()),
		widget.NewLabelWithStyle("Accounts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	d.accountList = widget.NewList(
		func() int {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.accounts)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.AccountIcon())
			name := widget.NewLabel("Account Placeholder")
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			if id < len(d.accounts) {
				box := obj.(*fyne.Container)
				lbl := box.Objects[0].(*widget.Label)
				acc := d.accounts[id]
				selectedMark := ""
				if d.selectedAccount != nil && d.selectedAccount.ID == acc.ID {
					selectedMark = " ● "
				}
				lbl.SetText(fmt.Sprintf("%s%s (%s)", selectedMark, acc.Name, acc.Type))
			}
		},
	)
	d.accountList.OnSelected = func(id widget.ListItemID) {
		d.mu.RLock()
		if id < len(d.accounts) {
			acc := d.accounts[id]
			d.mu.RUnlock()
			d.selectAccount(acc)
		} else {
			d.mu.RUnlock()
		}
	}

	createBucketBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
		d.showCreateBucketDialog()
	})
	deleteBucketBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		d.showDeleteBucketDialog()
	})

	d.bucketTabBadge = widget.NewLabelWithStyle("(0)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	d.bucketSearch = widget.NewEntry()
	d.bucketSearch.SetPlaceHolder("Filter buckets...")
	d.bucketSearch.SetIcon(theme.SearchIcon())
	d.bucketSearch.OnChanged = func(q string) {
		d.mu.Lock()
		d.bucketFilter = strings.TrimSpace(q)
		d.mu.Unlock()
		if d.bucketList != nil {
			d.bucketList.Refresh()
		}
	}

	bucketHeader := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				widget.NewIcon(theme.StorageIcon()),
				widget.NewLabelWithStyle("Buckets", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				d.bucketTabBadge,
			),
			container.NewHBox(createBucketBtn, deleteBucketBtn),
		),
		d.bucketSearch,
	)

	d.bucketList = widget.NewList(
		func() int {
			return len(d.getFilteredBuckets())
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.StorageIcon())
			name := widget.NewLabel("Bucket Placeholder")
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			filtered := d.getFilteredBuckets()
			d.mu.RLock()
			selectedName := d.selectedBucket
			d.mu.RUnlock()
			if id < len(filtered) {
				box := obj.(*fyne.Container)
				lbl := box.Objects[0].(*widget.Label)
				b := filtered[id]
				selectedMark := ""
				if b.Name == selectedName {
					selectedMark = "▶ "
				}
				regText := ""
				if b.Region != "" {
					regText = fmt.Sprintf(" [%s]", b.Region)
				}
				lbl.SetText(fmt.Sprintf("%s%s%s", selectedMark, b.Name, regText))
			}
		},
	)
	d.bucketList.OnSelected = func(id widget.ListItemID) {
		filtered := d.getFilteredBuckets()
		if id < len(filtered) {
			bName := filtered[id].Name
			d.selectBucket(bName)
		}
	}

	accountsBox := container.NewBorder(accountHeader, nil, nil, nil, d.accountList)
	bucketsBox := container.NewBorder(bucketHeader, nil, nil, nil, d.bucketList)

	accTab := container.NewTabItemWithIcon("Accounts", theme.AccountIcon(), accountsBox)
	bucketTab := container.NewTabItemWithIcon("Buckets", theme.StorageIcon(), bucketsBox)
	tabs := container.NewAppTabs(accTab, bucketTab)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.SelectIndex(1)

	return tabs
}

func (d *DesktopApp) buildCenterPanel() fyne.CanvasObject {
	uploadFileBtn := widget.NewButtonWithIcon("Upload File", theme.UploadIcon(), func() {
		d.showUploadDialog()
	})
	uploadFileBtn.Importance = widget.HighImportance

	uploadFolderBtn := widget.NewButtonWithIcon("Upload Folder", theme.FolderNewIcon(), func() {
		d.showUploadFolderDialog()
	})

	newFolderBtn := widget.NewButtonWithIcon("New Folder", theme.FolderNewIcon(), func() {
		d.showCreateFolderDialog()
	})

	previewTopBtn := widget.NewButtonWithIcon("Preview", theme.VisibilityIcon(), func() {
		d.previewSelectedObject()
	})

	downloadBtn := widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
		d.downloadSelectedObject()
	})

	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		d.deleteSelectedObject()
	})
	deleteBtn.Importance = widget.DangerImportance

	d.openFolderBtn = widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		d.openSelectedFolder()
	})
	d.openFolderBtn.Disable()

	d.viewModeSelect = widget.NewSelect([]string{"Hierarchical (Folders)", "Flat (All Keys)"}, func(selected string) {
		d.setViewMode(selected == "Flat (All Keys)")
	})
	d.viewModeSelect.SetSelected("Hierarchical (Folders)")

	actionsBar := container.NewHBox(
		uploadFileBtn,
		uploadFolderBtn,
		newFolderBtn,
		widget.NewSeparator(),
		d.openFolderBtn,
		previewTopBtn,
		downloadBtn,
		deleteBtn,
		widget.NewSeparator(),
		d.viewModeSelect,
	)

	d.upFolderBtn = widget.NewButtonWithIcon("⬆ Up", theme.NavigateBackIcon(), func() {
		d.navigateUp()
	})
	d.upFolderBtn.Importance = widget.HighImportance
	d.upFolderBtn.Disable()

	d.prefixNavLabel = widget.NewLabel("Prefix: /")
	d.prefixNavLabel.TextStyle = fyne.TextStyle{Monospace: true}
	d.prefixNavLabel.Truncation = fyne.TextTruncateEllipsis

	toggleInspectorBtn := widget.NewButtonWithIcon("Inspector", theme.MenuIcon(), func() {
		d.toggleInspector()
	})
	toggleInspectorBtn.Importance = widget.LowImportance

	navBar := container.NewBorder(
		nil,
		nil,
		d.upFolderBtn,
		toggleInspectorBtn,
		d.prefixNavLabel,
	)

	topControls := container.NewVBox(
		actionsBar,
		navBar,
		widget.NewSeparator(),
	)

	d.objectTable = widget.NewTable(
		func() (int, int) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.filteredObjects) + 1, 4
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.FileIcon())
			lbl := widget.NewLabel("Cell text placeholder")
			lbl.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, icon, nil, lbl)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			box := cell.(*fyne.Container)
			lbl := box.Objects[0].(*widget.Label)
			icon := box.Objects[1].(*widget.Icon)

			if id.Row == 0 {
				icon.Hide()
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				switch id.Col {
				case 0:
					lbl.SetText("Name")
				case 1:
					lbl.SetText("Size")
				case 2:
					lbl.SetText("Type / Storage Class")
				case 3:
					lbl.SetText("Last Modified")
				}
				return
			}

			lbl.TextStyle = fyne.TextStyle{Bold: false}
			d.mu.RLock()
			defer d.mu.RUnlock()
			idx := id.Row - 1
			if idx < 0 || idx >= len(d.filteredObjects) {
				icon.Hide()
				lbl.SetText("")
				return
			}
			obj := d.filteredObjects[idx]
			switch id.Col {
			case 0:
				if obj.IsPrefix {
					icon.SetResource(theme.FolderIcon())
					icon.Show()
					displayKey := obj.Key
					if !d.flatMode && d.currentPrefix != "" {
						displayKey = strings.TrimPrefix(displayKey, d.currentPrefix)
					}
					lbl.SetText(displayKey)
				} else {
					icon.SetResource(theme.FileIcon())
					icon.Show()
					displayKey := obj.Key
					if !d.flatMode && d.currentPrefix != "" {
						displayKey = strings.TrimPrefix(displayKey, d.currentPrefix)
					}
					lbl.SetText(displayKey)
				}
			case 1:
				icon.Hide()
				if obj.IsPrefix {
					lbl.SetText("-")
				} else {
					lbl.SetText(formatBytes(obj.Size))
				}
			case 2:
				icon.Hide()
				if obj.IsPrefix {
					lbl.SetText("📁 Folder")
				} else {
					class := string(obj.StorageClass)
					if class == "" {
						class = "STANDARD"
					}
					lbl.SetText(class)
				}
			case 3:
				icon.Hide()
				if obj.IsPrefix {
					lbl.SetText("-")
				} else {
					lbl.SetText(obj.LastModified.Format("2006-01-02 15:04:05"))
				}
			}
		},
	)

	d.objectTable.SetColumnWidth(0, 360)
	d.objectTable.SetColumnWidth(1, 110)
	d.objectTable.SetColumnWidth(2, 160)
	d.objectTable.SetColumnWidth(3, 190)

	d.objectTable.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			return
		}
		d.handleTableRowSelected(id.Row-1, id.Col)
	}

	d.loadingBar = widget.NewProgressBarInfinite()
	d.loadingBar.Stop()
	d.loadingLabel = widget.NewLabel("Loading objects...")
	d.loadingLabel.TextStyle = fyne.TextStyle{Italic: true}
	d.loadingContainer = container.NewCenter(
		container.NewVBox(
			d.loadingLabel,
			container.NewGridWrap(fyne.NewSize(240, 10), d.loadingBar),
		),
	)
	d.loadingContainer.Hide()

	d.emptyStateMsg = widget.NewLabel("This bucket is empty.\nUse 'Upload File' or 'Upload Folder' to add items.")
	d.emptyStateMsg.Wrapping = fyne.TextWrapWord
	d.emptyUploadBtn = widget.NewButtonWithIcon("Upload File", theme.UploadIcon(), func() {
		d.showUploadDialog()
	})
	d.emptyUploadBtn.Importance = widget.HighImportance

	d.emptyUploadDirBtn = widget.NewButtonWithIcon("Upload Folder", theme.FolderNewIcon(), func() {
		d.showUploadFolderDialog()
	})

	d.emptyClearFilterBtn = widget.NewButtonWithIcon("Clear Filter", theme.CancelIcon(), func() {
		if d.searchEntry != nil {
			d.searchEntry.SetText("")
		}
	})
	d.emptyClearFilterBtn.Hide()

	emptyActionsBox := container.NewHBox(d.emptyUploadBtn, d.emptyUploadDirBtn, d.emptyClearFilterBtn)
	d.emptyStateCard = widget.NewCard(
		"No Objects Found",
		"",
		container.NewVBox(
			d.emptyStateMsg,
			emptyActionsBox,
		),
	)
	d.emptyStateCard.Hide()

	d.centerContainer = container.NewStack(
		d.objectTable,
		d.emptyStateCard,
		d.loadingContainer,
	)

	return container.NewBorder(topControls, nil, nil, nil, d.centerContainer)
}

func (d *DesktopApp) toggleInspector() {
	if d.inspectorSplit == nil {
		return
	}
	d.inspectorOpen = !d.inspectorOpen
	if d.inspectorOpen {
		d.inspectorSplit.SetOffset(0.70)
	} else {
		d.inspectorSplit.SetOffset(1.0)
	}
}

func (d *DesktopApp) buildRightPanel() fyne.CanvasObject {
	d.metadataLabel = widget.NewLabel("Select an object to inspect details.")
	d.metadataLabel.Wrapping = fyne.TextWrapWord

	d.presignedLabel = widget.NewEntry()
	d.presignedLabel.SetPlaceHolder("Presigned URL will show here")

	genURLBtn := widget.NewButtonWithIcon("Generate (1h)", theme.MediaPlayIcon(), func() {
		d.generatePresignedURL()
	})

	copyPresignedBtn := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		if d.presignedLabel.Text != "" {
			d.window.Clipboard().SetContent(d.presignedLabel.Text)
			d.showToast("Presigned URL copied to clipboard")
		}
	})

	presignedBox := container.NewBorder(
		nil,
		nil,
		genURLBtn,
		copyPresignedBtn,
		d.presignedLabel,
	)

	copyURIBtn := widget.NewButtonWithIcon("Copy S3 URI", theme.ContentCopyIcon(), func() {
		d.mu.RLock()
		b := d.selectedBucket
		obj := d.selectedObject
		d.mu.RUnlock()
		if b != "" && obj != nil {
			uri := fmt.Sprintf("s3://%s/%s", b, obj.Key)
			d.window.Clipboard().SetContent(uri)
			d.showToast("S3 URI copied: " + uri)
		}
	})

	actionPreviewBtn := widget.NewButtonWithIcon("Preview Content", theme.VisibilityIcon(), func() {
		d.previewSelectedObject()
	})

	actionDownloadBtn := widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
		d.downloadSelectedObject()
	})

	actionDeleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		d.deleteSelectedObject()
	})
	actionDeleteBtn.Importance = widget.DangerImportance

	quickActions := container.NewVBox(
		widget.NewLabelWithStyle("Quick Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(actionPreviewBtn, copyURIBtn),
		container.NewHBox(actionDownloadBtn, actionDeleteBtn),
	)

	d.tagsLabel = widget.NewLabel("Tags: None")
	d.tagsLabel.Wrapping = fyne.TextWrapWord

	tagsAccordion := widget.NewAccordion(
		widget.NewAccordionItem("Metadata & Tags", d.tagsLabel),
	)

	d.previewBtn = widget.NewButtonWithIcon("Preview Object", theme.VisibilityIcon(), func() {
		d.previewSelectedObject()
	})

	d.openExternalBtn = widget.NewButtonWithIcon("Open External", theme.MediaPlayIcon(), func() {
		d.openSelectedExternal()
	})
	d.openExternalBtn.Disable()

	d.previewStatusLabel = widget.NewLabel("Click 'Preview Object' or double-click to view content.")
	d.previewStatusLabel.Wrapping = fyne.TextWrapWord
	d.previewStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	d.previewContentEntry = widget.NewMultiLineEntry()
	d.previewContentEntry.Wrapping = fyne.TextWrapWord
	d.previewContentEntry.SetPlaceHolder("Object content preview will appear here...")
	d.previewContentEntry.Disable()

	previewScroll := container.NewGridWrap(fyne.NewSize(240, 160), d.previewContentEntry)

	d.previewImage = canvas.NewImageFromResource(theme.FileIcon())
	d.previewImage.FillMode = canvas.ImageFillContain
	d.previewImage.SetMinSize(fyne.NewSize(240, 160))
	d.previewImageWrap = container.NewGridWrap(fyne.NewSize(240, 160), d.previewImage)
	d.previewImageWrap.Hide()

	d.previewVideoBadge = widget.NewLabelWithStyle("🎬 Video detected", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	playVideoBtn := widget.NewButtonWithIcon("▶ Open Video in System Player", theme.MediaPlayIcon(), func() {
		d.openSelectedExternal()
	})
	playVideoBtn.Importance = widget.HighImportance
	d.previewVideoBox = container.NewVBox(
		d.previewVideoBadge,
		playVideoBtn,
	)
	d.previewVideoBox.Hide()

	previewContainer := container.NewVBox(
		container.NewHBox(d.previewBtn, d.openExternalBtn),
		d.previewStatusLabel,
		d.previewImageWrap,
		d.previewVideoBox,
		previewScroll,
	)

	detailsContent := container.NewVBox(
		widget.NewLabelWithStyle("File Summary", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.metadataLabel,
		widget.NewSeparator(),
		quickActions,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Content Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		previewContainer,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Presigned URL", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		presignedBox,
		widget.NewSeparator(),
		tagsAccordion,
	)

	d.detailsCard = widget.NewCard("Object Inspector", "Metadata, actions & preview", container.NewVScroll(detailsContent))
	return d.detailsCard
}

func (d *DesktopApp) buildBottomPanel() fyne.CanvasObject {
	statusTitle := widget.NewLabelWithStyle("Transfers & Activity", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	d.transferStatus = widget.NewLabel("Idle")
	d.transferStatus.TextStyle = fyne.TextStyle{Italic: true}

	clearCompletedBtn := widget.NewButtonWithIcon("Clear Completed", theme.ContentClearIcon(), func() {
		d.clearCompletedTransfers()
	})
	clearCompletedBtn.Importance = widget.LowImportance

	toggleTransfersBtn := widget.NewButtonWithIcon("Toggle Tray", theme.MenuDropDownIcon(), func() {
		d.toggleTransfersTray()
	})
	toggleTransfersBtn.Importance = widget.LowImportance

	topBar := container.NewBorder(
		nil,
		nil,
		container.NewHBox(widget.NewIcon(theme.HistoryIcon()), statusTitle, d.transferStatus),
		container.NewHBox(clearCompletedBtn, toggleTransfersBtn),
	)

	d.transferList = widget.NewList(
		func() int {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.jobs)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.UploadIcon())
			nameLabel := widget.NewLabel("Job Name")
			statusLabel := widget.NewLabel("Status")
			progress := widget.NewProgressBar()
			leftBox := container.NewHBox(icon, nameLabel)
			return container.NewBorder(nil, nil, leftBox, statusLabel, progress)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			if id < len(d.jobs) {
				j := d.jobs[id]
				border := obj.(*fyne.Container)
				leftBox := border.Objects[1].(*fyne.Container)
				icon := leftBox.Objects[0].(*widget.Icon)
				nameLbl := leftBox.Objects[1].(*widget.Label)
				statusLbl := border.Objects[2].(*widget.Label)
				pBar := border.Objects[0].(*widget.ProgressBar)

				if j.Type == transferDomain.TransferTypeUpload {
					icon.SetResource(theme.UploadIcon())
				} else {
					icon.SetResource(theme.DownloadIcon())
				}

				nameLbl.SetText(fmt.Sprintf("[%s] %s -> %s", j.Type, filepath.Base(j.SourcePath), filepath.Base(j.DestinationPath)))
				statusLbl.SetText(fmt.Sprintf("%s (%s / %s)", j.Status, formatBytes(j.BytesTransferred), formatBytes(j.TotalBytes)))

				if j.TotalBytes > 0 {
					pBar.SetValue(float64(j.BytesTransferred) / float64(j.TotalBytes))
				} else {
					pBar.SetValue(0)
				}
			}
		},
	)

	scrollTransfers := container.NewScroll(d.transferList)
	trayContent := container.NewGridWrap(fyne.NewSize(1200, 110), scrollTransfers)

	d.transfersTray = container.NewVBox(
		topBar,
		trayContent,
	)

	return container.NewVBox(
		widget.NewSeparator(),
		d.transfersTray,
	)
}

func (d *DesktopApp) toggleTransfersTray() {
	if d.transfersTray == nil {
		return
	}
	d.transfersOpen = !d.transfersOpen
	if len(d.transfersTray.Objects) >= 2 {
		if d.transfersOpen {
			d.transfersTray.Objects[1].Show()
		} else {
			d.transfersTray.Objects[1].Hide()
		}
		d.transfersTray.Refresh()
	}
}

func (d *DesktopApp) clearCompletedTransfers() {
	d.mu.Lock()
	var remaining []transferDomain.TransferJob
	for _, j := range d.jobs {
		if j.Status == transferDomain.JobStatusRunning || j.Status == transferDomain.JobStatusPending {
			remaining = append(remaining, j)
		}
	}
	d.jobs = remaining
	d.mu.Unlock()

	if d.transferList != nil {
		d.transferList.Refresh()
	}
	d.showToast("Cleared completed transfers")
}

func (d *DesktopApp) loadAccounts() {
	ctx := context.Background()
	accs, err := d.container.AccountService.ListAccounts(ctx)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	d.mu.Lock()
	d.accounts = accs
	names := make([]string, 0, len(accs))
	for _, a := range accs {
		names = append(names, a.Name)
	}
	d.mu.Unlock()

	d.accountSelect.SetOptions(names)
	if d.accountList != nil {
		d.accountList.Refresh()
	}

	if len(accs) > 0 && d.selectedAccount == nil {
		d.selectAccount(accs[0])
	}
}

func (d *DesktopApp) onAccountSelectedByName(name string) {
	d.mu.RLock()
	var found *accountDomain.Account
	for _, a := range d.accounts {
		if a.Name == name {
			found = a
			break
		}
	}
	d.mu.RUnlock()

	if found != nil {
		d.selectAccount(found)
	}
}

func (d *DesktopApp) selectAccount(acc *accountDomain.Account) {
	d.mu.Lock()
	if d.selectedAccount != nil && d.selectedAccount.ID == acc.ID {
		d.mu.Unlock()
		return
	}
	d.selectedAccount = acc
	d.buckets = nil
	d.selectedBucket = ""
	d.prefixes = nil
	d.objects = nil
	d.filteredObjects = nil
	d.selectedObject = nil
	d.currentPrefix = ""
	d.lastSelectedRow = -1
	d.lastSelectedCol = -1
	d.mu.Unlock()

	if d.objectTable != nil {
		d.objectTable.UnselectAll()
	}
	if d.accountSelect.Selected != acc.Name {
		d.accountSelect.SetSelected(acc.Name)
	}
	if d.accountList != nil {
		d.accountList.Refresh()
	}
	d.bucketBadge.SetText("No Bucket Selected")
	if d.endpointBadge != nil {
		ep := acc.Region
		if acc.Endpoint != "" {
			ep += fmt.Sprintf(" (%s)", acc.Endpoint)
		}
		if ep == "" {
			ep = "global"
		}
		d.endpointBadge.SetText(fmt.Sprintf("[%s]", ep))
	}

	d.updateBreadcrumbs()
	d.loadBuckets()
	d.loadTransfers()
	d.showToast(fmt.Sprintf("Switched to profile '%s'", acc.Name))
}

func (d *DesktopApp) loadBuckets() {
	d.mu.RLock()
	acc := d.selectedAccount
	d.mu.RUnlock()

	if acc == nil {
		return
	}

	ctx := context.Background()
	bService, err := d.container.CreateBucketService(ctx, acc.Name)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	buckets, err := bService.ListBuckets(ctx)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	d.mu.Lock()
	d.buckets = buckets
	count := len(buckets)
	d.mu.Unlock()

	if d.bucketTabBadge != nil {
		d.bucketTabBadge.SetText(fmt.Sprintf("(%d)", count))
	}

	if d.bucketList != nil {
		d.bucketList.Refresh()
	}

	if len(buckets) > 0 && d.selectedBucket == "" {
		d.selectBucket(buckets[0].Name)
	}
}

func (d *DesktopApp) selectBucket(name string) {
	d.mu.Lock()
	d.selectedBucket = name
	d.bucketBadge.SetText(name)
	d.prefixes = nil
	d.objects = nil
	d.filteredObjects = nil
	d.selectedObject = nil
	d.currentPrefix = ""
	d.lastSelectedRow = -1
	d.lastSelectedCol = -1
	d.mu.Unlock()

	if d.objectTable != nil {
		d.objectTable.UnselectAll()
	}
	if d.bucketList != nil {
		d.bucketList.Refresh()
	}

	d.updatePrefixNavUI()
	d.updateBreadcrumbs()
	d.selectObject(nil)
	d.loadObjects()
	d.showToast(fmt.Sprintf("Opened bucket '%s'", name))
}

func (d *DesktopApp) loadObjects() {
	d.mu.Lock()
	if d.loadCancel != nil {
		d.loadCancel()
		d.loadCancel = nil
	}
	d.loadSeq++
	currentSeq := d.loadSeq

	acc := d.selectedAccount
	bucket := d.selectedBucket
	prefix := d.currentPrefix
	isFlat := d.flatMode

	if acc == nil || bucket == "" {
		d.prefixes = nil
		d.objects = nil
		d.filteredObjects = nil
		d.mu.Unlock()
		d.updateTableViewState(false, 0)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.loadCancel = cancel
	d.mu.Unlock()

	d.updateTableViewState(true, 0)

	go func() {
		defer cancel()

		oService, err := d.container.CreateObjectService(ctx, acc.Name)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			fyne.Do(func() {
				d.mu.Lock()
				if d.loadSeq != currentSeq {
					d.mu.Unlock()
					return
				}
				d.mu.Unlock()
				d.updateTableViewState(false, 0)
				dialog.ShowError(err, d.window)
			})
			return
		}

		var allObjects []objectDomain.Object
		var allPrefixes []objectDomain.Prefix
		continuation := ""
		const maxBatchKeys = 1000
		const maxTotalKeys = 50000

		delimiter := "/"
		if isFlat {
			delimiter = ""
		}

		for {
			if ctx.Err() != nil {
				return
			}

			res, err := oService.ListObjects(ctx, bucket, objectDomain.ObjectFilter{
				Prefix:       prefix,
				Delimiter:    delimiter,
				MaxKeys:      maxBatchKeys,
				Continuation: continuation,
			})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				fyne.Do(func() {
					d.mu.Lock()
					if d.loadSeq != currentSeq {
						d.mu.Unlock()
						return
					}
					d.mu.Unlock()
					d.updateTableViewState(false, len(allObjects)+len(allPrefixes))
					dialog.ShowError(err, d.window)
				})
				return
			}

			allObjects = append(allObjects, res.Objects...)
			if !isFlat && len(res.CommonPrefixes) > 0 {
				allPrefixes = append(allPrefixes, res.CommonPrefixes...)
			}

			loadedCount := len(allObjects) + len(allPrefixes)
			if res.IsTruncated && res.NextContinuationToken != "" && loadedCount < maxTotalKeys {
				continuation = res.NextContinuationToken
				fyne.Do(func() {
					d.mu.Lock()
					if d.loadSeq == currentSeq && d.loadingLabel != nil {
						d.loadingLabel.SetText(fmt.Sprintf("Loading items (%d loaded)...", loadedCount))
					}
					d.mu.Unlock()
				})
			} else {
				break
			}
		}

		fyne.Do(func() {
			d.mu.Lock()
			if d.loadSeq != currentSeq {
				d.mu.Unlock()
				return
			}
			d.prefixes = allPrefixes
			d.objects = allObjects
			d.applyFilterLocked()
			totalFiltered := len(d.filteredObjects)
			d.mu.Unlock()

			d.updatePrefixNavUI()
			d.updateBreadcrumbs()
			d.updateTableViewState(false, totalFiltered)
		})
	}()
}

func (d *DesktopApp) updateTableViewState(loading bool, itemCount int) {
	if d.loadingContainer == nil || d.objectTable == nil || d.emptyStateCard == nil {
		return
	}

	if loading {
		if d.loadingBar != nil {
			d.loadingBar.Start()
		}
		if d.loadingLabel != nil {
			d.loadingLabel.SetText("Loading objects...")
		}
		d.loadingContainer.Show()
		d.emptyStateCard.Hide()
		d.objectTable.Hide()
	} else {
		if d.loadingBar != nil {
			d.loadingBar.Stop()
		}
		d.loadingContainer.Hide()
		if itemCount == 0 {
			d.mu.RLock()
			query := d.searchQuery
			d.mu.RUnlock()
			if query != "" {
				if d.emptyStateMsg != nil {
					d.emptyStateMsg.SetText(fmt.Sprintf("No objects match query '%s'.\nTry clearing search or checking prefix.", query))
				}
				if d.emptyClearFilterBtn != nil {
					d.emptyClearFilterBtn.Show()
				}
				if d.emptyUploadBtn != nil {
					d.emptyUploadBtn.Hide()
				}
				if d.emptyUploadDirBtn != nil {
					d.emptyUploadDirBtn.Hide()
				}
			} else {
				if d.emptyStateMsg != nil {
					d.emptyStateMsg.SetText("This bucket is empty.\nUse 'Upload File' or 'Upload Folder' to add items.")
				}
				if d.emptyClearFilterBtn != nil {
					d.emptyClearFilterBtn.Hide()
				}
				if d.emptyUploadBtn != nil {
					d.emptyUploadBtn.Show()
				}
				if d.emptyUploadDirBtn != nil {
					d.emptyUploadDirBtn.Show()
				}
			}
			d.emptyStateCard.Show()
			d.objectTable.Hide()
		} else {
			d.emptyStateCard.Hide()
			d.objectTable.Show()
			d.objectTable.Refresh()
		}
	}
}

func (d *DesktopApp) onSearchChanged(query string) {
	d.mu.Lock()
	d.searchQuery = strings.TrimSpace(strings.ToLower(query))
	d.applyFilterLocked()
	total := len(d.filteredObjects)
	d.mu.Unlock()

	d.updateTableViewState(false, total)
}

func (d *DesktopApp) applyFilterLocked() {
	var combined []objectDomain.Object

	for _, p := range d.prefixes {
		combined = append(combined, objectDomain.Object{
			Bucket:   p.Bucket,
			Key:      p.Prefix,
			IsPrefix: true,
		})
	}
	combined = append(combined, d.objects...)

	if d.searchQuery == "" {
		d.filteredObjects = combined
		return
	}

	filtered := make([]objectDomain.Object, 0)
	for _, item := range combined {
		if strings.Contains(strings.ToLower(item.Key), d.searchQuery) {
			filtered = append(filtered, item)
		}
	}
	d.filteredObjects = filtered
}

func (d *DesktopApp) updatePrefixNavUI() {
	d.mu.RLock()
	prefix := d.currentPrefix
	isFlat := d.flatMode
	d.mu.RUnlock()

	if d.prefixNavLabel != nil {
		if isFlat {
			d.prefixNavLabel.SetText("Prefix: (Flat listing - Delimiter disabled)")
		} else if prefix == "" {
			d.prefixNavLabel.SetText("Prefix: / (root)")
		} else {
			d.prefixNavLabel.SetText(fmt.Sprintf("Prefix: /%s", prefix))
		}
	}

	if d.upFolderBtn != nil {
		if !isFlat && prefix != "" {
			d.upFolderBtn.Enable()
		} else {
			d.upFolderBtn.Disable()
		}
	}
}

func (d *DesktopApp) navigateToPrefix(prefix string) {
	d.mu.Lock()
	d.currentPrefix = prefix
	d.prefixes = nil
	d.objects = nil
	d.filteredObjects = nil
	d.selectedObject = nil
	d.lastSelectedRow = -1
	d.lastSelectedCol = -1
	d.mu.Unlock()

	if d.objectTable != nil {
		d.objectTable.UnselectAll()
	}

	d.updatePrefixNavUI()
	d.updateBreadcrumbs()
	d.selectObject(nil)
	d.loadObjects()
}

func (d *DesktopApp) navigateUp() {
	d.mu.RLock()
	prefix := d.currentPrefix
	isFlat := d.flatMode
	d.mu.RUnlock()

	if isFlat || prefix == "" {
		return
	}

	trimmed := strings.TrimSuffix(prefix, "/")
	lastSlash := strings.LastIndex(trimmed, "/")
	parent := ""
	if lastSlash >= 0 {
		parent = trimmed[:lastSlash+1]
	}

	d.navigateToPrefix(parent)
}

func (d *DesktopApp) openSelectedFolder() {
	d.mu.RLock()
	selected := d.selectedObject
	d.mu.RUnlock()

	if selected == nil || !selected.IsPrefix {
		return
	}

	target := selected.Key
	if !strings.HasSuffix(target, "/") {
		target += "/"
	}

	d.navigateToPrefix(target)
}

func (d *DesktopApp) setViewMode(flat bool) {
	d.mu.Lock()
	if d.flatMode == flat {
		d.mu.Unlock()
		return
	}
	d.flatMode = flat
	d.prefixes = nil
	d.objects = nil
	d.filteredObjects = nil
	d.selectedObject = nil
	d.lastSelectedRow = -1
	d.lastSelectedCol = -1
	d.mu.Unlock()

	if d.objectTable != nil {
		d.objectTable.UnselectAll()
	}

	d.updatePrefixNavUI()
	d.updateBreadcrumbs()
	d.selectObject(nil)
	d.loadObjects()
}

func (d *DesktopApp) handleTableRowSelected(idx int, col int) {
	d.mu.Lock()
	if idx < 0 || idx >= len(d.filteredObjects) {
		d.mu.Unlock()
		return
	}

	obj := d.filteredObjects[idx]
	now := time.Now()
	isDoubleClick := (d.lastSelectedRow == idx) && (d.lastSelectedCol == col) && (now.Sub(d.lastSelectedTime) < 500*time.Millisecond)
	d.lastSelectedRow = idx
	d.lastSelectedCol = col
	d.lastSelectedTime = now
	d.mu.Unlock()

	d.selectObject(&obj)

	if isDoubleClick {
		if obj.IsPrefix {
			d.openSelectedFolder()
		} else {
			d.previewSelectedObject()
		}
	}
}

func (d *DesktopApp) selectObject(obj *objectDomain.Object) {
	d.mu.Lock()
	if d.inspectCancel != nil {
		d.inspectCancel()
		d.inspectCancel = nil
	}
	d.selectedObject = obj
	d.mu.Unlock()

	if d.openFolderBtn != nil {
		if obj != nil && obj.IsPrefix {
			d.openFolderBtn.Enable()
		} else {
			d.openFolderBtn.Disable()
		}
	}

	if obj == nil {
		if d.metadataLabel != nil {
			d.metadataLabel.SetText("Select an object to inspect details.")
		}
		if d.presignedLabel != nil {
			d.presignedLabel.SetText("")
		}
		if d.tagsLabel != nil {
			d.tagsLabel.SetText("Tags: None")
		}
		if d.previewStatusLabel != nil {
			d.previewStatusLabel.SetText("Click 'Preview Object' or double-click to view content.")
		}
		if d.previewContentEntry != nil {
			d.previewContentEntry.SetText("")
		}
		if d.previewImageWrap != nil {
			d.previewImageWrap.Hide()
		}
		if d.previewVideoBox != nil {
			d.previewVideoBox.Hide()
		}
		if d.openExternalBtn != nil {
			d.openExternalBtn.Disable()
		}
		return
	}

	if obj.IsPrefix {
		if d.metadataLabel != nil {
			d.metadataLabel.SetText(fmt.Sprintf("Directory: %s\nType: Virtual Common Prefix / Directory\nClick 'Open Folder' or double-click to navigate inside.", obj.Key))
		}
		if d.presignedLabel != nil {
			d.presignedLabel.SetText("")
		}
		if d.tagsLabel != nil {
			d.tagsLabel.SetText("Tags: None (Directory)")
		}
		if d.previewStatusLabel != nil {
			d.previewStatusLabel.SetText("Directory: open folder to explore contents.")
		}
		if d.previewContentEntry != nil {
			d.previewContentEntry.SetText("")
		}
		if d.previewImageWrap != nil {
			d.previewImageWrap.Hide()
		}
		if d.previewVideoBox != nil {
			d.previewVideoBox.Hide()
		}
		if d.openExternalBtn != nil {
			d.openExternalBtn.Disable()
		}
		return
	}

	class := string(obj.StorageClass)
	if class == "" {
		class = "STANDARD"
	}
	details := fmt.Sprintf(
		"Name: %s\nKey: %s\nSize: %s (%d bytes)\nStorage Class: %s\nETag: %s\nLast Modified: %s",
		filepath.Base(obj.Key),
		obj.Key,
		formatBytes(obj.Size),
		obj.Size,
		class,
		obj.ETag,
		obj.LastModified.Format("2006-01-02 15:04:05 UTC"),
	)
	if d.metadataLabel != nil {
		d.metadataLabel.SetText(details)
	}
	if d.presignedLabel != nil {
		d.presignedLabel.SetText("")
	}
	if d.openExternalBtn != nil {
		d.openExternalBtn.Enable()
	}
	if d.previewImageWrap != nil {
		d.previewImageWrap.Hide()
	}
	if d.previewVideoBox != nil {
		if isVideoFile(obj.Key) {
			if d.previewVideoBadge != nil {
				d.previewVideoBadge.SetText(fmt.Sprintf("🎬 Video: %s (%s)", filepath.Base(obj.Key), formatBytes(obj.Size)))
			}
			d.previewVideoBox.Show()
		} else {
			d.previewVideoBox.Hide()
		}
	}
	if d.previewStatusLabel != nil {
		if isImageFile(obj.Key) {
			d.previewStatusLabel.SetText(fmt.Sprintf("Image detected: %s (%s). Click Preview to render.", filepath.Base(obj.Key), formatBytes(obj.Size)))
		} else if isVideoFile(obj.Key) {
			d.previewStatusLabel.SetText(fmt.Sprintf("Video detected: %s (%s). Click 'Open Video' to play in system player.", filepath.Base(obj.Key), formatBytes(obj.Size)))
		} else {
			d.previewStatusLabel.SetText(fmt.Sprintf("Ready to preview %s (%s)", filepath.Base(obj.Key), formatBytes(obj.Size)))
		}
	}
	if d.previewContentEntry != nil {
		d.previewContentEntry.SetText("")
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.mu.Lock()
	d.inspectCancel = cancel
	d.mu.Unlock()
	go d.fetchObjectDetailsAndTags(ctx, obj.Key)
}

func (d *DesktopApp) fetchObjectDetailsAndTags(ctx context.Context, key string) {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	selected := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || selected == nil || selected.Key != key {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	oService, err := d.container.CreateObjectService(ctx, acc.Name)
	if err != nil {
		return
	}

	meta, err := oService.GetObjectMetadata(ctx, bucket, key, "")
	if err == nil {
		fyne.Do(func() {
			d.mu.RLock()
			cur := d.selectedObject
			d.mu.RUnlock()
			if cur != nil && cur.Key == key {
				class := string(cur.StorageClass)
				if class == "" {
					class = "STANDARD"
				}
				cType := meta.ContentType
				if cType == "" {
					cType = "application/octet-stream"
				}
				details := fmt.Sprintf(
					"Name: %s\nKey: %s\nSize: %s (%d bytes)\nStorage Class: %s\nContent-Type: %s\nETag: %s\nLast Modified: %s",
					filepath.Base(cur.Key),
					cur.Key,
					formatBytes(cur.Size),
					cur.Size,
					class,
					cType,
					cur.ETag,
					cur.LastModified.Format("2006-01-02 15:04:05 UTC"),
				)
				if d.metadataLabel != nil {
					d.metadataLabel.SetText(details)
				}
			}
		})
	}

	tags, err := oService.GetObjectTags(ctx, bucket, key, "")
	if err != nil {
		return
	}

	fyne.Do(func() {
		d.mu.RLock()
		cur := d.selectedObject
		d.mu.RUnlock()
		if cur == nil || cur.Key != key {
			return
		}

		if len(tags) == 0 {
			if d.tagsLabel != nil {
				d.tagsLabel.SetText("Tags: None")
			}
			return
		}
		var sb strings.Builder
		for _, t := range tags {
			sb.WriteString(fmt.Sprintf("%s = %s\n", t.Key, t.Value))
		}
		if d.tagsLabel != nil {
			d.tagsLabel.SetText(strings.TrimRight(sb.String(), "\n"))
		}
	})
}

func isImageFile(key string) bool {
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

func isVideoFile(key string) bool {
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".mp4", ".mkv", ".webm", ".avi", ".mov", ".flv":
		return true
	default:
		return false
	}
}

func openFileWithSystemApp(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd.exe", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func (d *DesktopApp) openSelectedExternal() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	obj := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || obj == nil {
		dialog.ShowInformation("Select Object", "Please select an object to open externally.", d.window)
		return
	}
	if obj.IsPrefix {
		d.openSelectedFolder()
		return
	}

	key := obj.Key
	d.showToast(fmt.Sprintf("Downloading %s to open externally...", filepath.Base(key)))

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		oService, err := d.container.CreateObjectService(ctx, acc.Name)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, d.window)
			})
			return
		}

		content, err := oService.GetObject(ctx, bucket, key, "")
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, d.window)
			})
			return
		}
		defer content.Body.Close()

		ext := filepath.Ext(key)
		tmpFile, err := os.CreateTemp("", "kumokura-open-*"+ext)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("create temp file: %w", err), d.window)
			})
			return
		}
		tmpPath := tmpFile.Name()

		_, copyErr := io.Copy(tmpFile, content.Body)
		_ = tmpFile.Close()
		if copyErr != nil {
			_ = os.Remove(tmpPath)
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("write temp file: %w", copyErr), d.window)
			})
			return
		}

		if err := openFileWithSystemApp(tmpPath); err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("open file in external viewer: %w", err), d.window)
			})
			return
		}

		fyne.Do(func() {
			d.showToast("Opened " + filepath.Base(key) + " in external viewer")
		})
	}()
}

func (d *DesktopApp) previewSelectedObject() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	obj := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || obj == nil {
		dialog.ShowInformation("Select Object", "Please select an object to preview.", d.window)
		return
	}

	if obj.IsPrefix {
		d.openSelectedFolder()
		return
	}

	key := obj.Key
	if d.previewStatusLabel != nil {
		d.previewStatusLabel.SetText("Fetching preview...")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		oService, err := d.container.CreateObjectService(ctx, acc.Name)
		if err != nil {
			fyne.Do(func() {
				if d.previewStatusLabel != nil {
					d.previewStatusLabel.SetText("Failed to initialize object service")
				}
				dialog.ShowError(err, d.window)
			})
			return
		}

		content, err := oService.GetObject(ctx, bucket, key, "")
		if err != nil {
			fyne.Do(func() {
				if d.previewStatusLabel != nil {
					d.previewStatusLabel.SetText("Failed to fetch object content")
				}
				dialog.ShowError(err, d.window)
			})
			return
		}
		defer content.Body.Close()

		if isImageFile(key) {
			const maxImageBytes = 10 * 1024 * 1024
			imgData, readErr := io.ReadAll(io.LimitReader(content.Body, maxImageBytes))
			if readErr != nil {
				fyne.Do(func() {
					if d.previewStatusLabel != nil {
						d.previewStatusLabel.SetText("Failed to read image data")
					}
					dialog.ShowError(readErr, d.window)
				})
				return
			}

			statusDesc := fmt.Sprintf("Image (%s): %s rendered", formatBytes(int64(len(imgData))), filepath.Base(key))
			res := fyne.NewStaticResource(filepath.Base(key), imgData)

			fyne.Do(func() {
				d.mu.RLock()
				cur := d.selectedObject
				d.mu.RUnlock()
				if cur != nil && cur.Key == key {
					if d.previewStatusLabel != nil {
						d.previewStatusLabel.SetText(statusDesc)
					}
					if d.previewImage != nil {
						d.previewImage.Resource = res
						d.previewImage.Refresh()
					}
					if d.previewImageWrap != nil {
						d.previewImageWrap.Show()
					}
					if d.previewContentEntry != nil {
						d.previewContentEntry.SetText("")
					}
				}
				d.showImagePreviewDialog(key, res)
			})
			return
		}

		if isVideoFile(key) {
			statusDesc := fmt.Sprintf("Video (%s): %s", formatBytes(obj.Size), filepath.Base(key))
			fyne.Do(func() {
				d.mu.RLock()
				cur := d.selectedObject
				d.mu.RUnlock()
				if cur != nil && cur.Key == key {
					if d.previewStatusLabel != nil {
						d.previewStatusLabel.SetText(statusDesc)
					}
					if d.previewVideoBox != nil {
						if d.previewVideoBadge != nil {
							d.previewVideoBadge.SetText(fmt.Sprintf("🎬 Video: %s (%s)", filepath.Base(key), formatBytes(obj.Size)))
						}
						d.previewVideoBox.Show()
					}
				}
				d.showVideoPreviewDialog(key, obj.Size)
			})
			return
		}

		const maxPreviewBytes = 64 * 1024
		buf := make([]byte, maxPreviewBytes)
		n, _ := io.ReadFull(content.Body, buf)
		snippet := buf[:n]

		isBinary := false
		for _, b := range snippet {
			if b == 0 {
				isBinary = true
				break
			}
		}

		var previewText string
		var statusDesc string

		if isBinary {
			statusDesc = fmt.Sprintf("Binary file (%s). Showing hex snippet of first %d bytes:", formatBytes(obj.Size), n)
			var hexLines strings.Builder
			for i := 0; i < n && i < 512; i += 16 {
				end := i + 16
				if end > n {
					end = n
				}
				chunk := snippet[i:end]
				hexPart := ""
				asciiPart := ""
				for _, b := range chunk {
					hexPart += fmt.Sprintf("%02X ", b)
					if b >= 32 && b <= 126 {
						asciiPart += string(b)
					} else {
						asciiPart += "."
					}
				}
				for len(hexPart) < 48 {
					hexPart += "   "
				}
				hexLines.WriteString(fmt.Sprintf("%04X: %s |%s|\n", i, hexPart, asciiPart))
			}
			previewText = hexLines.String()
		} else {
			statusDesc = fmt.Sprintf("Text file (%s). Previewing first %d bytes:", formatBytes(obj.Size), n)
			cleanText := strings.ReplaceAll(string(snippet), "\x00", "")
			previewText = cleanText
		}

		fyne.Do(func() {
			d.mu.RLock()
			cur := d.selectedObject
			d.mu.RUnlock()
			if cur != nil && cur.Key == key {
				if d.previewStatusLabel != nil {
					d.previewStatusLabel.SetText(statusDesc)
				}
				if d.previewImageWrap != nil {
					d.previewImageWrap.Hide()
				}
				if d.previewVideoBox != nil {
					d.previewVideoBox.Hide()
				}
				if d.previewContentEntry != nil {
					d.previewContentEntry.SetText(previewText)
				}
			}
			d.showPreviewDialog(key, statusDesc, previewText)
		})
	}()
}

func (d *DesktopApp) showImagePreviewDialog(key string, res fyne.Resource) {
	title := fmt.Sprintf("Image Preview: %s", filepath.Base(key))
	img := canvas.NewImageFromResource(res)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(640, 440))

	openExtBtn := widget.NewButtonWithIcon("Open in System Viewer", theme.MediaPlayIcon(), func() {
		d.openSelectedExternal()
	})
	openExtBtn.Importance = widget.HighImportance

	contentBox := container.NewBorder(
		nil,
		openExtBtn,
		nil,
		nil,
		container.NewGridWrap(fyne.NewSize(680, 460), img),
	)

	dModal := dialog.NewCustom(title, "Close", contentBox, d.window)
	dModal.Resize(fyne.NewSize(720, 560))
	dModal.Show()
}

func (d *DesktopApp) showVideoPreviewDialog(key string, size int64) {
	title := fmt.Sprintf("Video: %s", filepath.Base(key))
	infoLabel := widget.NewLabel(fmt.Sprintf("Format: %s\nSize: %s (%d bytes)\nKey: %s",
		strings.ToUpper(strings.TrimPrefix(filepath.Ext(key), ".")),
		formatBytes(size),
		size,
		key,
	))
	infoLabel.TextStyle = fyne.TextStyle{Monospace: true}

	playBtn := widget.NewButtonWithIcon("▶ Open Video in System Player", theme.MediaPlayIcon(), func() {
		d.openSelectedExternal()
	})
	playBtn.Importance = widget.HighImportance

	contentBox := container.NewVBox(
		widget.NewLabelWithStyle("Video File Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		infoLabel,
		widget.NewSeparator(),
		playBtn,
	)

	dModal := dialog.NewCustom(title, "Close", contentBox, d.window)
	dModal.Resize(fyne.NewSize(520, 260))
	dModal.Show()
}

func (d *DesktopApp) showPreviewDialog(key, statusDesc, previewText string) {
	title := fmt.Sprintf("Preview: %s", filepath.Base(key))
	descLabel := widget.NewLabel(statusDesc)
	descLabel.TextStyle = fyne.TextStyle{Italic: true}

	entry := widget.NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapWord
	entry.SetText(previewText)

	copyBtn := widget.NewButtonWithIcon("Copy Content", theme.ContentCopyIcon(), func() {
		d.window.Clipboard().SetContent(previewText)
		d.showToast("Preview content copied to clipboard")
	})

	openExtBtn := widget.NewButtonWithIcon("Open External", theme.MediaPlayIcon(), func() {
		d.openSelectedExternal()
	})

	bottomBar := container.NewHBox(copyBtn, openExtBtn)

	contentBox := container.NewBorder(
		descLabel,
		bottomBar,
		nil,
		nil,
		container.NewGridWrap(fyne.NewSize(680, 420), entry),
	)

	dModal := dialog.NewCustom(title, "Close", contentBox, d.window)
	dModal.Resize(fyne.NewSize(720, 520))
	dModal.Show()
}

func (d *DesktopApp) generatePresignedURL() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	obj := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || obj == nil {
		dialog.ShowInformation("No Object", "Please select an object first.", d.window)
		return
	}

	if obj.IsPrefix {
		dialog.ShowInformation("Directory", "Presigned URLs cannot be generated for directories.", d.window)
		return
	}

	ctx := context.Background()
	oService, err := d.container.CreateObjectService(ctx, acc.Name)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	pURL, err := oService.GeneratePresignedURL(ctx, bucket, obj.Key, "GET", time.Hour)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	d.presignedLabel.SetText(pURL.URL)
	d.showToast("Presigned URL generated (expires in 1h)")
}

func (d *DesktopApp) showCreateFolderDialog() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	d.mu.RUnlock()

	if acc == nil || bucket == "" {
		dialog.ShowInformation("Select Bucket", "Please select an account and bucket before creating folders.", d.window)
		return
	}

	entry := widget.NewEntry()
	entry.SetPlaceHolder("folder-name")

	dialog.ShowCustomConfirm("New Folder", "Create", "Cancel", container.NewVBox(
		widget.NewLabel("Enter folder name:"),
		entry,
	), func(confirmed bool) {
		if !confirmed {
			return
		}
		name := strings.TrimSpace(entry.Text)
		if name == "" {
			return
		}
		if !strings.HasSuffix(name, "/") {
			name += "/"
		}

		d.mu.RLock()
		fullKey := name
		if d.currentPrefix != "" && !d.flatMode {
			fullKey = d.currentPrefix + name
		}
		d.mu.RUnlock()

		go func() {
			ctx := context.Background()
			oService, err := d.container.CreateObjectService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			_, err = oService.PutObject(ctx, bucket, fullKey, bytes.NewReader([]byte{}), 0, objectDomain.ObjectMetadata{
				ContentType: "application/x-directory",
			})
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			d.showToast(fmt.Sprintf("Folder '%s' created", fullKey))
			d.loadObjects()
		}()
	}, d.window)
}

func (d *DesktopApp) showUploadDialog() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	d.mu.RUnlock()

	if acc == nil || bucket == "" {
		dialog.ShowInformation("Select Bucket", "Please select an account and bucket before uploading.", d.window)
		return
	}

	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, d.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		filePath := reader.URI().Path()
		key := filepath.Base(filePath)
		d.mu.RLock()
		if d.currentPrefix != "" && !d.flatMode {
			key = d.currentPrefix + key
		}
		d.mu.RUnlock()
		d.showToast(fmt.Sprintf("Starting upload of %s...", key))

		go func() {
			ctx := context.Background()
			oService, err := d.container.CreateObjectService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			f, err := os.Open(filePath)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}
			defer f.Close()

			stat, err := f.Stat()
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			totalSize := stat.Size()
			jobID := fmt.Sprintf("up-%d", time.Now().UnixNano())
			accountID := string(acc.ID)
			now := time.Now()
			job := transferDomain.TransferJob{
				ID:              jobID,
				AccountID:       accountID,
				Type:            transferDomain.TransferTypeUpload,
				SourcePath:      filePath,
				DestinationPath: fmt.Sprintf("s3://%s/%s", bucket, key),
				Bucket:          bucket,
				Key:             key,
				Status:          transferDomain.JobStatusRunning,
				TotalBytes:      totalSize,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if d.container.TransferRepo != nil {
				_ = d.container.TransferRepo.SaveJob(ctx, job)
				d.loadTransfers()
			}

			pr := &progressReader{
				reader: f,
				total:  totalSize,
				onProgress: func(transferred int64) {
					if d.container.TransferRepo != nil {
						_ = d.container.TransferRepo.UpdateJobProgress(ctx, jobID, transferred)
						d.loadTransfers()
					}
				},
			}

			_, err = oService.PutObject(ctx, bucket, key, pr, totalSize, objectDomain.ObjectMetadata{})
			if err != nil {
				if d.container.TransferRepo != nil {
					_ = d.container.TransferRepo.UpdateJobStatus(ctx, jobID, transferDomain.JobStatusFailed, err.Error())
					d.loadTransfers()
				}
				dialog.ShowError(err, d.window)
				return
			}

			if d.container.TransferRepo != nil {
				_ = d.container.TransferRepo.UpdateJobProgress(ctx, jobID, totalSize)
				_ = d.container.TransferRepo.UpdateJobStatus(ctx, jobID, transferDomain.JobStatusCompleted, "")
				d.loadTransfers()
			}

			d.showToast(fmt.Sprintf("Uploaded %s successfully", key))
			d.loadObjects()
		}()
	}, d.window)
}

func (d *DesktopApp) showUploadFolderDialog() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	d.mu.RUnlock()

	if acc == nil || bucket == "" {
		dialog.ShowInformation("Select Bucket", "Please select an account and bucket before uploading.", d.window)
		return
	}

	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, d.window)
			return
		}
		if uri == nil {
			return
		}

		folderPath := uri.Path()
		d.showToast(fmt.Sprintf("Uploading folder %s...", filepath.Base(folderPath)))

		go func() {
			ctx := context.Background()
			concurrency := d.container.Config.MaxUploadConcurrency
			if concurrency <= 0 {
				concurrency = 100
			}
			if fc, _ := config.LoadFolderConfig(folderPath); fc != nil && fc.MaxConcurrency > 0 {
				concurrency = fc.MaxConcurrency
			}

			oService, err := d.container.CreateObjectService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			folderBase := filepath.Base(folderPath)
			d.mu.RLock()
			targetFolderPrefix := filepath.ToSlash(filepath.Clean(d.currentPrefix))
			flat := d.flatMode
			d.mu.RUnlock()

			if targetFolderPrefix == "." {
				targetFolderPrefix = ""
			}
			if targetFolderPrefix != "" && !strings.HasSuffix(targetFolderPrefix, "/") {
				targetFolderPrefix += "/"
			}
			folderS3Prefix := targetFolderPrefix + folderBase + "/"
			if flat {
				folderS3Prefix = folderBase + "/"
			}

			_, _ = oService.PutObject(ctx, bucket, folderS3Prefix, bytes.NewReader([]byte{}), 0, objectDomain.ObjectMetadata{
				ContentType: "application/x-directory",
			})

			var filesToUpload []string
			var emptyDirsToUpload []string

			_ = filepath.Walk(folderPath, func(p string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if info.IsDir() {
					if p != folderPath {
						entries, readErr := os.ReadDir(p)
						if readErr == nil && len(entries) == 0 {
							emptyDirsToUpload = append(emptyDirsToUpload, p)
						}
					}
					return nil
				}
				filesToUpload = append(filesToUpload, p)
				return nil
			})

			for _, dirPath := range emptyDirsToUpload {
				rel, err := filepath.Rel(folderPath, dirPath)
				if err != nil {
					continue
				}
				dirKey := folderS3Prefix + filepath.ToSlash(rel) + "/"
				_, _ = oService.PutObject(ctx, bucket, dirKey, bytes.NewReader([]byte{}), 0, objectDomain.ObjectMetadata{
					ContentType: "application/x-directory",
				})
			}

			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup
			var uploadCount int64
			accountID := string(acc.ID)

			for _, filePath := range filesToUpload {
				rel, err := filepath.Rel(folderPath, filePath)
				if err != nil {
					continue
				}
				key := folderS3Prefix + filepath.ToSlash(rel)

				st, sErr := os.Stat(filePath)
				if sErr != nil {
					continue
				}
				fileSize := st.Size()
				jobID := fmt.Sprintf("up-%d", time.Now().UnixNano())
				now := time.Now()

				job := transferDomain.TransferJob{
					ID:              jobID,
					AccountID:       accountID,
					Type:            transferDomain.TransferTypeUpload,
					SourcePath:      filePath,
					DestinationPath: fmt.Sprintf("s3://%s/%s", bucket, key),
					Bucket:          bucket,
					Key:             key,
					Status:          transferDomain.JobStatusRunning,
					TotalBytes:      fileSize,
					CreatedAt:       now,
					UpdatedAt:       now,
				}
				if d.container.TransferRepo != nil {
					_ = d.container.TransferRepo.SaveJob(ctx, job)
					d.loadTransfers()
				}

				sem <- struct{}{}
				wg.Add(1)
				go func(fPath, objKey, jID string, totalBytes int64) {
					defer func() {
						<-sem
						wg.Done()
					}()

					f, oErr := os.Open(fPath)
					if oErr != nil {
						if d.container.TransferRepo != nil {
							_ = d.container.TransferRepo.UpdateJobStatus(ctx, jID, transferDomain.JobStatusFailed, oErr.Error())
							d.loadTransfers()
						}
						return
					}
					defer f.Close()

					pr := &progressReader{
						reader: f,
						total:  totalBytes,
						onProgress: func(transferred int64) {
							if d.container.TransferRepo != nil {
								_ = d.container.TransferRepo.UpdateJobProgress(ctx, jID, transferred)
								d.loadTransfers()
							}
						},
					}

					_, pErr := oService.PutObject(ctx, bucket, objKey, pr, totalBytes, objectDomain.ObjectMetadata{})
					if pErr != nil {
						if d.container.TransferRepo != nil {
							_ = d.container.TransferRepo.UpdateJobStatus(ctx, jID, transferDomain.JobStatusFailed, pErr.Error())
							d.loadTransfers()
						}
						return
					}

					if d.container.TransferRepo != nil {
						_ = d.container.TransferRepo.UpdateJobProgress(ctx, jID, totalBytes)
						_ = d.container.TransferRepo.UpdateJobStatus(ctx, jID, transferDomain.JobStatusCompleted, "")
						d.loadTransfers()
					}
					atomic.AddInt64(&uploadCount, 1)
				}(filePath, key, jobID, fileSize)
			}
			wg.Wait()

			d.showToast(fmt.Sprintf("Folder upload complete: %d files to %s", uploadCount, bucket))
			d.loadObjects()
		}()
	}, d.window)
}

func (d *DesktopApp) showSettingsDialog() {
	concurrencyEntry := widget.NewEntry()
	concurrencyEntry.SetText(fmt.Sprintf("%d", d.container.Config.MaxUploadConcurrency))

	items := []*widget.FormItem{
		widget.NewFormItem("Upload Concurrency (Goroutines)", concurrencyEntry),
	}

	dialog.ShowForm("Preferences", "Save", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		var val int
		if _, err := fmt.Sscanf(concurrencyEntry.Text, "%d", &val); err == nil && val > 0 {
			d.container.Config.MaxUploadConcurrency = val
			_ = d.container.Config.Save()
			if d.container.TransferService != nil {
				d.container.TransferService.SetMaxConcurrency(val)
			}
			d.showToast("Settings updated")
		}
	}, d.window)
}

func (d *DesktopApp) downloadSelectedObject() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	obj := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || obj == nil {
		dialog.ShowInformation("Select Object", "Please select an object to download.", d.window)
		return
	}

	if obj.IsPrefix {
		dialog.ShowInformation("Directory", "Cannot download a directory directly. Please open it to download individual files.", d.window)
		return
	}

	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, d.window)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		d.showToast(fmt.Sprintf("Starting download of %s...", obj.Key))

		go func() {
			ctx := context.Background()
			oService, err := d.container.CreateObjectService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			content, err := oService.GetObject(ctx, bucket, obj.Key, "")
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}
			defer content.Body.Close()

			jobID := fmt.Sprintf("dl-%d", time.Now().UnixNano())
			accountID := string(acc.ID)
			now := time.Now()
			destPath := writer.URI().Path()

			job := transferDomain.TransferJob{
				ID:              jobID,
				AccountID:       accountID,
				Type:            transferDomain.TransferTypeDownload,
				SourcePath:      fmt.Sprintf("s3://%s/%s", bucket, obj.Key),
				DestinationPath: destPath,
				Bucket:          bucket,
				Key:             obj.Key,
				Status:          transferDomain.JobStatusRunning,
				TotalBytes:      obj.Size,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if d.container.TransferRepo != nil {
				_ = d.container.TransferRepo.SaveJob(ctx, job)
				d.loadTransfers()
			}

			pr := &progressReader{
				reader: content.Body,
				total:  obj.Size,
				onProgress: func(transferred int64) {
					if d.container.TransferRepo != nil {
						_ = d.container.TransferRepo.UpdateJobProgress(ctx, jobID, transferred)
						d.loadTransfers()
					}
				},
			}

			_, err = io.Copy(writer, pr)
			if err != nil {
				if d.container.TransferRepo != nil {
					_ = d.container.TransferRepo.UpdateJobStatus(ctx, jobID, transferDomain.JobStatusFailed, err.Error())
					d.loadTransfers()
				}
				dialog.ShowError(err, d.window)
				return
			}

			if d.container.TransferRepo != nil {
				_ = d.container.TransferRepo.UpdateJobProgress(ctx, jobID, obj.Size)
				_ = d.container.TransferRepo.UpdateJobStatus(ctx, jobID, transferDomain.JobStatusCompleted, "")
				d.loadTransfers()
			}
			d.showToast(fmt.Sprintf("Downloaded %s successfully", obj.Key))
		}()
	}, d.window)
}

func (d *DesktopApp) deleteSelectedObject() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	obj := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || obj == nil {
		dialog.ShowInformation("Select Object", "Please select an object to delete.", d.window)
		return
	}

	if obj.IsPrefix {
		dialog.ShowInformation("Directory", "Deleting common prefix directories directly is not supported.", d.window)
		return
	}

	targetKey := obj.Key
	confirmPrompt := container.NewVBox(
		widget.NewLabel("Are you sure you want to permanently delete:"),
		widget.NewLabelWithStyle(targetKey, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("This action cannot be undone."),
	)

	dialog.ShowCustomConfirm("Confirm Delete", "Delete", "Cancel", confirmPrompt, func(ok bool) {
		if !ok {
			return
		}
		go func() {
			ctx := context.Background()
			oService, err := d.container.CreateObjectService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			err = oService.DeleteObject(ctx, bucket, targetKey, "")
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			d.selectObject(nil)
			d.showToast(fmt.Sprintf("Deleted '%s'", targetKey))
			d.loadObjects()
		}()
	}, d.window)
}

func (d *DesktopApp) showCreateBucketDialog() {
	d.mu.RLock()
	acc := d.selectedAccount
	d.mu.RUnlock()

	if acc == nil {
		dialog.ShowInformation("Select Account", "Please select an account first.", d.window)
		return
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Bucket Name")
	regionEntry := widget.NewEntry()
	regionEntry.SetText(acc.Region)

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Region", regionEntry),
	}

	dialog.ShowForm("Create Bucket", "Create", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		bucketName := strings.TrimSpace(nameEntry.Text)
		region := strings.TrimSpace(regionEntry.Text)
		if bucketName == "" {
			return
		}

		go func() {
			ctx := context.Background()
			bService, err := d.container.CreateBucketService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			err = bService.CreateBucket(ctx, bucketName, region)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			d.showToast(fmt.Sprintf("Bucket '%s' created", bucketName))
			d.loadBuckets()
		}()
	}, d.window)
}

func (d *DesktopApp) showDeleteBucketDialog() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	d.mu.RUnlock()

	if acc == nil || bucket == "" {
		dialog.ShowInformation("Select Bucket", "Please select a bucket to delete.", d.window)
		return
	}

	confirmPrompt := container.NewVBox(
		widget.NewLabel("Are you sure you want to delete bucket:"),
		widget.NewLabelWithStyle(bucket, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("All contained objects must already be deleted."),
	)

	dialog.ShowCustomConfirm("Confirm Delete Bucket", "Delete Bucket", "Cancel", confirmPrompt, func(ok bool) {
		if !ok {
			return
		}
		go func() {
			ctx := context.Background()
			bService, err := d.container.CreateBucketService(ctx, acc.Name)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			err = bService.DeleteBucket(ctx, bucket)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			d.mu.Lock()
			d.selectedBucket = ""
			d.mu.Unlock()
			d.bucketBadge.SetText("No Bucket Selected")
			d.updateBreadcrumbs()
			d.showToast(fmt.Sprintf("Bucket '%s' deleted", bucket))
			d.loadBuckets()
		}()
	}, d.window)
}

func (d *DesktopApp) showAddAccountDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Account Name")

	typeSelect := widget.NewSelect([]string{"AWS", "CloudflareR2", "Wasabi", "MinIO", "Custom"}, nil)
	typeSelect.SetSelected("AWS")

	endpointEntry := widget.NewEntry()
	endpointEntry.SetPlaceHolder("Optional custom endpoint (e.g. localhost:9000)")

	regionEntry := widget.NewEntry()
	regionEntry.SetText("us-east-1")

	accessKeyEntry := widget.NewEntry()
	accessKeyEntry.SetPlaceHolder("Access Key ID")

	secretKeyEntry := widget.NewPasswordEntry()
	secretKeyEntry.SetPlaceHolder("Secret Access Key")

	pathStyleCheck := widget.NewCheck("Use Path Style", nil)

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Provider Type", typeSelect),
		widget.NewFormItem("Endpoint", endpointEntry),
		widget.NewFormItem("Region", regionEntry),
		widget.NewFormItem("Access Key ID", accessKeyEntry),
		widget.NewFormItem("Secret Access Key", secretKeyEntry),
		widget.NewFormItem("Path Style", pathStyleCheck),
	}

	dialog.ShowForm("Add Storage Account", "Save", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}

		creds, err := accountDomain.NewCredentials(
			strings.TrimSpace(accessKeyEntry.Text),
			strings.TrimSpace(secretKeyEntry.Text),
			"",
		)
		if err != nil {
			dialog.ShowError(err, d.window)
			return
		}

		ctx := context.Background()
		_, err = d.container.AccountService.CreateAccount(ctx, accountPorts.CreateAccountParams{
			Name:         strings.TrimSpace(nameEntry.Text),
			Type:         accountDomain.AccountType(typeSelect.Selected),
			Endpoint:     strings.TrimSpace(endpointEntry.Text),
			Region:       strings.TrimSpace(regionEntry.Text),
			UsePathStyle: pathStyleCheck.Checked,
			Credentials:  creds,
		})
		if err != nil {
			dialog.ShowError(err, d.window)
			return
		}

		d.showToast(fmt.Sprintf("Added account '%s'", strings.TrimSpace(nameEntry.Text)))
		d.loadAccounts()
	}, d.window)
}

func (d *DesktopApp) refreshAll() {
	d.showToast("Refreshing accounts, buckets, and objects...")
	d.loadAccounts()
	d.loadBuckets()
	d.loadObjects()
	d.loadTransfers()
}

func (d *DesktopApp) loadTransfers() {
	d.mu.RLock()
	acc := d.selectedAccount
	d.mu.RUnlock()

	var accountID string
	if acc != nil {
		accountID = string(acc.ID)
	}

	ctx := context.Background()
	jobs, err := d.container.TransferRepo.ListJobs(ctx, accountID, "")
	if err != nil {
		return
	}

	d.mu.Lock()
	d.jobs = jobs
	activeCount := 0
	for _, j := range jobs {
		if j.Status == transferDomain.JobStatusRunning || j.Status == transferDomain.JobStatusPending {
			activeCount++
		}
	}
	d.mu.Unlock()

	if d.transferStatus != nil {
		if activeCount > 0 {
			d.transferStatus.SetText(fmt.Sprintf("%d active transfer(s)", activeCount))
		} else {
			d.transferStatus.SetText("Idle")
		}
	}

	if d.transferList != nil {
		d.transferList.Refresh()
	}
}

func (d *DesktopApp) startBackgroundPoller() {
	d.refreshTicker = time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-d.refreshTicker.C:
				d.loadTransfers()
			case <-d.stopTicker:
				d.refreshTicker.Stop()
				return
			}
		}
	}()
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
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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
