package desktop

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	accountDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	accountPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/sekai-labs/kumokura/internal/bootstrap"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objectDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
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

	objects         []objectDomain.Object
	filteredObjects []objectDomain.Object
	selectedObject  *objectDomain.Object

	currentPrefix string
	searchQuery   string

	jobs []transferDomain.TransferJob

	accountSelect *widget.Select
	bucketBadge   *widget.Label
	searchEntry   *widget.Entry

	accountList *widget.List
	bucketList  *widget.List

	objectTable *widget.Table

	detailsCard    *widget.Card
	metadataLabel  *widget.Label
	presignedLabel *widget.Entry
	tagsLabel      *widget.Label

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
		refreshBtn,
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
	uploadBtn := widget.NewButtonWithIcon("Upload", theme.UploadIcon(), func() {
		d.showUploadDialog()
	})
	downloadBtn := widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
		d.downloadSelectedObject()
	})
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		d.deleteSelectedObject()
	})

	actionsBar := container.NewHBox(
		uploadBtn,
		downloadBtn,
		deleteBtn,
	)

	tableHeader := container.NewBorder(
		nil,
		nil,
		widget.NewLabelWithStyle("Objects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		actionsBar,
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
				lbl.SetText(obj.Key)
			case 1:
				lbl.SetText(formatBytes(obj.Size))
			case 2:
				lbl.SetText(string(obj.StorageClass))
			case 3:
				lbl.SetText(obj.LastModified.Format("2006-01-02 15:04:05"))
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
		d.mu.RLock()
		idx := id.Row - 1
		if idx >= 0 && idx < len(d.filteredObjects) {
			obj := d.filteredObjects[idx]
			d.mu.RUnlock()
			d.selectObject(&obj)
		} else {
			d.mu.RUnlock()
		}
	}

	return container.NewBorder(tableHeader, nil, nil, nil, d.objectTable)
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

	detailsContent := container.NewVBox(
		widget.NewLabelWithStyle("Metadata", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.metadataLabel,
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
	d.objects = nil
	d.filteredObjects = nil
	d.selectedObject = nil
	d.mu.Unlock()

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
	d.objects = nil
	d.filteredObjects = nil
	d.selectedObject = nil
	d.mu.Unlock()

	d.loadObjects()
}

func (d *DesktopApp) loadObjects() {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	prefix := d.currentPrefix
	d.mu.RUnlock()

	if acc == nil || bucket == "" {
		return
	}

	ctx := context.Background()
	oService, err := d.container.CreateObjectService(ctx, acc.Name)
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	res, err := oService.ListObjects(ctx, bucket, objectDomain.ObjectFilter{
		Prefix:    prefix,
		Delimiter: "/",
		MaxKeys:   1000,
	})
	if err != nil {
		dialog.ShowError(err, d.window)
		return
	}

	d.mu.Lock()
	d.objects = res.Objects
	d.applyFilterLocked()
	d.mu.Unlock()

	if d.objectTable != nil {
		d.objectTable.Refresh()
	}
}

func (d *DesktopApp) onSearchChanged(query string) {
	d.mu.Lock()
	d.searchQuery = strings.TrimSpace(strings.ToLower(query))
	d.applyFilterLocked()
	d.mu.Unlock()

	if d.objectTable != nil {
		d.objectTable.Refresh()
	}
}

func (d *DesktopApp) applyFilterLocked() {
	if d.searchQuery == "" {
		d.filteredObjects = d.objects
		return
	}

	filtered := make([]objectDomain.Object, 0)
	for _, obj := range d.objects {
		if strings.Contains(strings.ToLower(obj.Key), d.searchQuery) {
			filtered = append(filtered, obj)
		}
	}
	d.filteredObjects = filtered
}

func (d *DesktopApp) selectObject(obj *objectDomain.Object) {
	d.mu.Lock()
	d.selectedObject = obj
	d.mu.Unlock()

	if obj == nil {
		d.metadataLabel.SetText("Select an object to inspect details.")
		d.presignedLabel.SetText("")
		d.tagsLabel.SetText("Tags: None")
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
	d.metadataLabel.SetText(details)
	d.presignedLabel.SetText("")

	go d.fetchObjectTags(obj.Key)
}

func (d *DesktopApp) fetchObjectTags(key string) {
	d.mu.RLock()
	acc := d.selectedAccount
	bucket := d.selectedBucket
	d.mu.RUnlock()

	if acc == nil || bucket == "" {
		return
	}

	ctx := context.Background()
	oService, err := d.container.CreateObjectService(ctx, acc.Name)
	if err != nil {
		return
	}

	tags, err := oService.GetObjectTags(ctx, bucket, key, "")
	if err != nil {
		d.tagsLabel.SetText("Tags: (None or inaccessible)")
		return
	}

	if len(tags) == 0 {
		d.tagsLabel.SetText("Tags: None")
		return
	}

	var sb strings.Builder
	for _, t := range tags {
		sb.WriteString(fmt.Sprintf("%s = %s\n", t.Key, t.Value))
	}
	d.tagsLabel.SetText(sb.String())
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
