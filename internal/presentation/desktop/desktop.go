package desktop

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	prefixes        []objectDomain.Prefix
	objects         []objectDomain.Object
	filteredObjects []objectDomain.Object
	selectedObject  *objectDomain.Object

	currentPrefix    string
	flatMode         bool
	searchQuery      string
	lastSelectedRow  int
	lastSelectedTime time.Time
	jobs             []transferDomain.TransferJob

	accountSelect *widget.Select
	bucketBadge   *widget.Label
	searchEntry   *widget.Entry

	accountList *widget.List
	bucketList  *widget.List

	objectTable      *widget.Table
	emptyStateCard   *widget.Card
	loadingContainer *fyne.Container
	loadingBar       *widget.ProgressBarInfinite
	loadingLabel     *widget.Label
	centerContainer  *fyne.Container

	prefixNavLabel *widget.Label
	upFolderBtn    *widget.Button
	viewModeSelect *widget.Select
	openFolderBtn  *widget.Button
	loadCancel     context.CancelFunc
	loadSeq        uint64

	detailsCard    *widget.Card
	metadataLabel  *widget.Label
	presignedLabel *widget.Entry
	tagsLabel      *widget.Label
	previewBtn     *widget.Button

	previewContentEntry *widget.Entry
	previewStatusLabel  *widget.Label

	transferList *widget.List

	refreshTicker *time.Ticker
	stopTicker    chan struct{}
}

func NewDesktopApp(appContainer *bootstrap.AppContainer) *DesktopApp {
	return NewDesktopAppWithFyneApp(appContainer, app.NewWithID("com.sekai.kumokura"))
}

func NewDesktopAppWithFyneApp(appContainer *bootstrap.AppContainer, a fyne.App) *DesktopApp {
	w := a.NewWindow("Kumokura - Native Object Storage Explorer")
	w.Resize(fyne.NewSize(1280, 800))

	da := &DesktopApp{
		container:  appContainer,
		fyneApp:    a,
		window:     w,
		stopTicker: make(chan struct{}),
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
	d.accountSelect = widget.NewSelect([]string{}, func(selected string) {
		d.onAccountSelectedByName(selected)
	})
	d.accountSelect.PlaceHolder = "Select Account"

	addAccountBtn := widget.NewButtonWithIcon("Add Account", theme.ContentAddIcon(), func() {
		d.showAddAccountDialog()
	})

	d.bucketBadge = widget.NewLabel("No Bucket Selected")
	d.bucketBadge.TextStyle = fyne.TextStyle{Bold: true}

	d.searchEntry = widget.NewEntry()
	d.searchEntry.SetPlaceHolder("Search objects by key...")
	d.searchEntry.OnChanged = func(query string) {
		d.onSearchChanged(query)
	}

	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
		d.showSettingsDialog()
	})

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		d.refreshAll()
	})
	leftHeader := container.NewHBox(
		widget.NewLabelWithStyle("Account:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.accountSelect,
		addAccountBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Bucket:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.bucketBadge,
	)

	rightHeader := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(settingsBtn, refreshBtn),
		d.searchEntry,
	)

	return container.NewVBox(
		container.NewBorder(nil, nil, leftHeader, nil, rightHeader),
		widget.NewSeparator(),
	)
}

func (d *DesktopApp) buildMasterDetail() fyne.CanvasObject {
	leftPanel := d.buildLeftPanel()
	centerPanel := d.buildCenterPanel()
	rightPanel := d.buildRightPanel()

	innerSplit := container.NewHSplit(centerPanel, rightPanel)
	innerSplit.SetOffset(0.70)

	outerSplit := container.NewHSplit(leftPanel, innerSplit)
	outerSplit.SetOffset(0.22)

	return outerSplit
}

func (d *DesktopApp) buildLeftPanel() fyne.CanvasObject {
	accountHeader := container.NewHBox(
		widget.NewLabelWithStyle("Accounts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	d.accountList = widget.NewList(
		func() int {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.accounts)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Account Placeholder")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			if id < len(d.accounts) {
				obj.(*widget.Label).SetText(fmt.Sprintf("%s (%s)", d.accounts[id].Name, d.accounts[id].Type))
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

	createBucketBtn := widget.NewButtonWithIcon("Create", theme.ContentAddIcon(), func() {
		d.showCreateBucketDialog()
	})
	deleteBucketBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		d.showDeleteBucketDialog()
	})

	bucketHeader := container.NewBorder(
		nil,
		nil,
		widget.NewLabelWithStyle("Buckets", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(createBucketBtn, deleteBucketBtn),
	)

	d.bucketList = widget.NewList(
		func() int {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.buckets)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Bucket Placeholder")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			if id < len(d.buckets) {
				obj.(*widget.Label).SetText(d.buckets[id].Name)
			}
		},
	)
	d.bucketList.OnSelected = func(id widget.ListItemID) {
		d.mu.RLock()
		if id < len(d.buckets) {
			bName := d.buckets[id].Name
			d.mu.RUnlock()
			d.selectBucket(bName)
		} else {
			d.mu.RUnlock()
		}
	}

	accountsBox := container.NewBorder(accountHeader, nil, nil, nil, d.accountList)
	bucketsBox := container.NewBorder(bucketHeader, nil, nil, nil, d.bucketList)

	return container.NewVSplit(accountsBox, bucketsBox)
}

func (d *DesktopApp) buildCenterPanel() fyne.CanvasObject {
	uploadFileBtn := widget.NewButtonWithIcon("Upload File", theme.UploadIcon(), func() {
		d.showUploadDialog()
	})
	uploadFolderBtn := widget.NewButtonWithIcon("Upload Folder", theme.FolderNewIcon(), func() {
		d.showUploadFolderDialog()
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

	d.openFolderBtn = widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		d.openSelectedFolder()
	})
	d.openFolderBtn.Disable()

	d.viewModeSelect = widget.NewSelect([]string{"Hierarchical (Folders)", "Flat (All Keys)"}, func(selected string) {
		d.setViewMode(selected == "Flat (All Keys)")
	})
	d.viewModeSelect.SetSelected("Hierarchical (Folders)")

	actionsBar := container.NewHBox(
		d.viewModeSelect,
		widget.NewSeparator(),
		d.openFolderBtn,
		uploadFileBtn,
		uploadFolderBtn,
		previewTopBtn,
		downloadBtn,
		deleteBtn,
	)

	tableHeader := container.NewBorder(
		nil,
		nil,
		widget.NewLabelWithStyle("Objects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		actionsBar,
	)

	d.upFolderBtn = widget.NewButtonWithIcon("Up / Parent", theme.NavigateBackIcon(), func() {
		d.navigateUp()
	})
	d.upFolderBtn.Disable()

	d.prefixNavLabel = widget.NewLabel("Prefix: /")
	d.prefixNavLabel.TextStyle = fyne.TextStyle{Monospace: true}
	d.prefixNavLabel.Truncation = fyne.TextTruncateEllipsis

	navBar := container.NewBorder(
		nil,
		nil,
		d.upFolderBtn,
		nil,
		d.prefixNavLabel,
	)

	topControls := container.NewVBox(
		tableHeader,
		navBar,
	)
	d.objectTable = widget.NewTable(
		func() (int, int) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.filteredObjects) + 1, 4
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("Cell text placeholder")
			lbl.Truncation = fyne.TextTruncateEllipsis
			return lbl
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			lbl := cell.(*widget.Label)
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				switch id.Col {
				case 0:
					lbl.SetText("Key")
				case 1:
					lbl.SetText("Size")
				case 2:
					lbl.SetText("Storage Class")
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
				lbl.SetText("")
				return
			}
			obj := d.filteredObjects[idx]
			switch id.Col {
			case 0:
				if obj.IsPrefix {
					lbl.SetText("[DIR] " + obj.Key)
				} else {
					lbl.SetText(obj.Key)
				}
			case 1:
				if obj.IsPrefix {
					lbl.SetText("-")
				} else {
					lbl.SetText(formatBytes(obj.Size))
				}
			case 2:
				if obj.IsPrefix {
					lbl.SetText("Directory")
				} else {
					lbl.SetText(string(obj.StorageClass))
				}
			case 3:
				if obj.IsPrefix {
					lbl.SetText("-")
				} else {
					lbl.SetText(obj.LastModified.Format("2006-01-02 15:04:05"))
				}
			}
		},
	)

	d.objectTable.SetColumnWidth(0, 320)
	d.objectTable.SetColumnWidth(1, 100)
	d.objectTable.SetColumnWidth(2, 140)
	d.objectTable.SetColumnWidth(3, 180)

	d.objectTable.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			return
		}
		d.handleTableRowSelected(id.Row - 1)
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

	emptyMsg := widget.NewLabel("This bucket is empty.\nUse 'Upload File' or 'Upload Folder' to add items.")
	emptyMsg.Wrapping = fyne.TextWrapWord
	emptyUploadBtn := widget.NewButtonWithIcon("Upload File", theme.UploadIcon(), func() {
		d.showUploadDialog()
	})
	emptyUploadFolderBtn := widget.NewButtonWithIcon("Upload Folder", theme.FolderNewIcon(), func() {
		d.showUploadFolderDialog()
	})
	d.emptyStateCard = widget.NewCard(
		"No Objects Found",
		"",
		container.NewVBox(
			emptyMsg,
			container.NewHBox(emptyUploadBtn, emptyUploadFolderBtn),
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

func (d *DesktopApp) buildRightPanel() fyne.CanvasObject {
	d.metadataLabel = widget.NewLabel("Select an object to inspect details.")
	d.metadataLabel.Wrapping = fyne.TextWrapWord

	d.presignedLabel = widget.NewEntry()
	d.presignedLabel.SetPlaceHolder("Presigned URL will show here")

	genURLBtn := widget.NewButtonWithIcon("Generate Presigned URL (1h)", theme.MediaPlayIcon(), func() {
		d.generatePresignedURL()
	})

	d.tagsLabel = widget.NewLabel("Tags: None")
	d.tagsLabel.Wrapping = fyne.TextWrapWord

	d.previewBtn = widget.NewButtonWithIcon("Preview Object", theme.VisibilityIcon(), func() {
		d.previewSelectedObject()
	})

	d.previewStatusLabel = widget.NewLabel("Click 'Preview Object' or double-click to view content.")
	d.previewStatusLabel.Wrapping = fyne.TextWrapWord
	d.previewStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	d.previewContentEntry = widget.NewMultiLineEntry()
	d.previewContentEntry.Wrapping = fyne.TextWrapWord
	d.previewContentEntry.SetPlaceHolder("Object content preview will appear here...")
	d.previewContentEntry.Disable()

	previewScroll := container.NewGridWrap(fyne.NewSize(320, 200), d.previewContentEntry)

	detailsContent := container.NewVBox(
		widget.NewLabelWithStyle("Metadata", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.metadataLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Content Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(d.previewBtn),
		d.previewStatusLabel,
		previewScroll,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Presigned URL", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		genURLBtn,
		d.presignedLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Tags & Attributes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.tagsLabel,
	)

	d.detailsCard = widget.NewCard("Object Details", "Metadata and Inspector", container.NewVScroll(detailsContent))
	return d.detailsCard
}

func (d *DesktopApp) buildBottomPanel() fyne.CanvasObject {
	statusTitle := widget.NewLabelWithStyle("Transfers & Activity", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	d.transferList = widget.NewList(
		func() int {
			d.mu.RLock()
			defer d.mu.RUnlock()
			return len(d.jobs)
		},
		func() fyne.CanvasObject {
			nameLabel := widget.NewLabel("Job Name")
			statusLabel := widget.NewLabel("Status")
			progress := widget.NewProgressBar()
			return container.NewBorder(nil, nil, nameLabel, statusLabel, progress)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			d.mu.RLock()
			defer d.mu.RUnlock()
			if id < len(d.jobs) {
				j := d.jobs[id]
				border := obj.(*fyne.Container)
				nameLbl := border.Objects[1].(*widget.Label)
				statusLbl := border.Objects[2].(*widget.Label)
				pBar := border.Objects[0].(*widget.ProgressBar)

				nameLbl.SetText(fmt.Sprintf("[%s] %s -> %s", j.Type, j.SourcePath, j.DestinationPath))
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
	return container.NewVBox(
		widget.NewSeparator(),
		statusTitle,
		container.NewGridWrap(fyne.NewSize(1200, 110), scrollTransfers),
	)
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
	d.mu.Unlock()

	d.updatePrefixNavUI()
	if d.accountSelect.Selected != acc.Name {
		d.accountSelect.SetSelected(acc.Name)
	}
	d.bucketBadge.SetText("No Bucket Selected")
	d.loadBuckets()
	d.loadTransfers()
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
	d.mu.Unlock()

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
	d.mu.Unlock()

	d.updatePrefixNavUI()
	d.selectObject(nil)
	d.loadObjects()
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
	d.mu.Unlock()

	d.updatePrefixNavUI()
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
	d.mu.Unlock()

	d.updatePrefixNavUI()
	d.selectObject(nil)
	d.loadObjects()
}

func (d *DesktopApp) handleTableRowSelected(idx int) {
	d.mu.Lock()
	if idx < 0 || idx >= len(d.filteredObjects) {
		d.mu.Unlock()
		return
	}

	obj := d.filteredObjects[idx]
	now := time.Now()
	isDoubleClick := (d.lastSelectedRow == idx) && (now.Sub(d.lastSelectedTime) < 500*time.Millisecond)
	d.lastSelectedRow = idx
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
		return
	}

	details := fmt.Sprintf(
		"Key: %s\nSize: %s (%d bytes)\nStorage Class: %s\nLast Modified: %s\nETag: %s",
		obj.Key,
		formatBytes(obj.Size),
		obj.Size,
		obj.StorageClass,
		obj.LastModified.Format("2006-01-02 15:04:05 UTC"),
		obj.ETag,
	)
	if d.metadataLabel != nil {
		d.metadataLabel.SetText(details)
	}
	if d.presignedLabel != nil {
		d.presignedLabel.SetText("")
	}
	if d.previewStatusLabel != nil {
		d.previewStatusLabel.SetText(fmt.Sprintf("Ready to preview %s (%s)", filepath.Base(obj.Key), formatBytes(obj.Size)))
	}
	if d.previewContentEntry != nil {
		d.previewContentEntry.SetText("")
	}

	go d.fetchObjectDetailsAndTags(obj.Key)
}

func (d *DesktopApp) fetchObjectDetailsAndTags(key string) {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	selected := d.selectedObject
	d.mu.RUnlock()

	if acc == nil || bucket == "" || selected == nil || selected.Key != key {
		return
	}

	ctx := context.Background()
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
			if cur == nil || cur.Key != key {
				return
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Key: %s\n", cur.Key))
			sb.WriteString(fmt.Sprintf("Size: %s (%d bytes)\n", formatBytes(cur.Size), cur.Size))
			if meta.ContentType != "" {
				sb.WriteString(fmt.Sprintf("Content-Type: %s\n", meta.ContentType))
			}
			sb.WriteString(fmt.Sprintf("Storage Class: %s\n", cur.StorageClass))
			sb.WriteString(fmt.Sprintf("Last Modified: %s\n", cur.LastModified.Format("2006-01-02 15:04:05 UTC")))
			if cur.ETag != "" {
				sb.WriteString(fmt.Sprintf("ETag: %s\n", cur.ETag))
			}
			if meta.CacheControl != "" {
				sb.WriteString(fmt.Sprintf("Cache-Control: %s\n", meta.CacheControl))
			}
			if meta.ContentEncoding != "" {
				sb.WriteString(fmt.Sprintf("Content-Encoding: %s\n", meta.ContentEncoding))
			}
			if len(meta.UserMetadata) > 0 {
				sb.WriteString("User Metadata:\n")
				for k, v := range meta.UserMetadata {
					sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
				}
			}

			if d.metadataLabel != nil {
				d.metadataLabel.SetText(strings.TrimRight(sb.String(), "\n"))
			}
		})
	}

	tags, err := oService.GetObjectTags(ctx, bucket, key, "")
	fyne.Do(func() {
		d.mu.RLock()
		cur := d.selectedObject
		d.mu.RUnlock()
		if cur == nil || cur.Key != key {
			return
		}

		if err != nil {
			if d.tagsLabel != nil {
				d.tagsLabel.SetText("Tags: (None or inaccessible)")
			}
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
				if d.previewContentEntry != nil {
					d.previewContentEntry.SetText(previewText)
				}
			}
			d.showPreviewDialog(key, statusDesc, previewText)
		})
	}()
}

func (d *DesktopApp) showPreviewDialog(key, statusDesc, previewText string) {
	title := fmt.Sprintf("Preview: %s", filepath.Base(key))
	descLabel := widget.NewLabel(statusDesc)
	descLabel.TextStyle = fyne.TextStyle{Italic: true}

	entry := widget.NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapWord
	entry.SetText(previewText)

	contentBox := container.NewBorder(
		descLabel,
		nil,
		nil,
		nil,
		container.NewGridWrap(fyne.NewSize(680, 420), entry),
	)

	dModal := dialog.NewCustom(title, "Close", contentBox, d.window)
	dModal.Resize(fyne.NewSize(720, 500))
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

			_, err = oService.PutObject(ctx, bucket, key, f, stat.Size(), objectDomain.ObjectMetadata{})
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			dialog.ShowInformation("Upload Success", fmt.Sprintf("Uploaded %s successfully.", key), d.window)
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

			var filesToUpload []string
			_ = filepath.Walk(folderPath, func(p string, info os.FileInfo, walkErr error) error {
				if walkErr == nil && !info.IsDir() {
					filesToUpload = append(filesToUpload, p)
				}
				return nil
			})

			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup
			var uploadCount int64

			for _, filePath := range filesToUpload {
				rel, err := filepath.Rel(folderPath, filePath)
				if err != nil {
					continue
				}
				key := filepath.ToSlash(rel)
				d.mu.RLock()
				if d.currentPrefix != "" && !d.flatMode {
					key = d.currentPrefix + key
				}
				d.mu.RUnlock()
				sem <- struct{}{}
				wg.Add(1)
				go func(fPath, objKey string) {
					defer func() {
						<-sem
						wg.Done()
					}()

					f, oErr := os.Open(fPath)
					if oErr != nil {
						return
					}
					defer f.Close()

					st, sErr := f.Stat()
					if sErr != nil {
						return
					}

					_, pErr := oService.PutObject(ctx, bucket, objKey, f, st.Size(), objectDomain.ObjectMetadata{})
					if pErr == nil {
						atomic.AddInt64(&uploadCount, 1)
					}
				}(filePath, key)
			}
			wg.Wait()

			dialog.ShowInformation("Folder Upload Complete", fmt.Sprintf("Uploaded %d files to %s.", uploadCount, bucket), d.window)
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

			_, err = io.Copy(writer, content.Body)
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			dialog.ShowInformation("Download Complete", fmt.Sprintf("Downloaded %s successfully.", obj.Key), d.window)
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

	confirm := dialog.NewConfirm("Confirm Delete", fmt.Sprintf("Are you sure you want to delete '%s'?", obj.Key), func(ok bool) {
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

			err = oService.DeleteObject(ctx, bucket, obj.Key, "")
			if err != nil {
				dialog.ShowError(err, d.window)
				return
			}

			d.selectObject(nil)
			d.loadObjects()
		}()
	}, d.window)
	confirm.Show()
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

	confirm := dialog.NewConfirm("Confirm Delete", fmt.Sprintf("Are you sure you want to delete bucket '%s'?", bucket), func(ok bool) {
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
			d.loadBuckets()
		}()
	}, d.window)
	confirm.Show()
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

		d.loadAccounts()
	}, d.window)
}

func (d *DesktopApp) refreshAll() {
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
	d.mu.Unlock()

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
