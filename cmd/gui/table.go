package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// table column indices — keep in sync with buildTable's header labels.
const (
	colAuto = iota
	colDrive
	colRemote
	colStatus
	colActions
	colCount
)

func (rm *rcloneManager) buildTable() {
	rm.table = widget.NewTable(
		func() (int, int) { return len(rm.cfg.Mounts), colCount },
		func() fyne.CanvasObject { return container.NewStack() },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			rm.updateTableCell(id, cell.(*fyne.Container))
		},
	)
	rm.table.ShowHeaderRow = true
	rm.table.HideSeparators = true
	rm.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	rm.table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		headers := [colCount]string{"자동", "드라이브", "리모트(서브경로)", "상태", ""}
		o.(*widget.Label).SetText(headers[id.Col])
	}
	rm.table.SetColumnWidth(colAuto, 50)
	rm.table.SetColumnWidth(colDrive, 80)
	rm.table.SetColumnWidth(colRemote, 280)
	rm.table.SetColumnWidth(colStatus, 80)
	rm.table.SetColumnWidth(colActions, 200)
}

// updateTableCell fills in one cell. CreateCell can't know in advance
// which column a recycled template will be asked to render, so each
// helper (cellCheck/cellLabel/cellActionButtons) replaces the cell's
// content if it isn't already the right widget type.
func (rm *rcloneManager) updateTableCell(id widget.TableCellID, cell *fyne.Container) {
	if id.Row >= len(rm.cfg.Mounts) {
		return
	}
	m := rm.cfg.Mounts[id.Row]

	switch id.Col {
	case colAuto:
		check := rm.cellCheck(cell)
		check.SetChecked(m.AutoMount)
		check.OnChanged = func(checked bool) {
			m.AutoMount = checked
			rm.saveMount(m)
		}

	case colDrive:
		label := rm.cellLabel(cell)
		label.SetText(displayDrive(m.Drive))

	case colRemote:
		label := rm.cellLabel(cell)
		label.SetText(fmt.Sprintf("%s:%s", m.Remote, m.RemotePath))

	case colStatus:
		label := rm.cellLabel(cell)
		label.SetText(statusLabel(rm.isRunning(m.ID)))

	case colActions:
		toggle, editBtn, delBtn := rm.cellActionButtons(cell)
		running := rm.isRunning(m.ID)
		toggle.SetText(toggleLabel(running))
		toggle.OnTapped = func() {
			if running {
				rm.unmount(m.ID)
			} else {
				rm.mount(m)
			}
		}
		editBtn.OnTapped = func() { rm.showMountDialog(&m) }
		delBtn.OnTapped = func() { rm.confirmDelete(m) }
	}
}

func (rm *rcloneManager) cellCheck(cell *fyne.Container) *widget.Check {
	if len(cell.Objects) == 1 {
		if c, ok := cell.Objects[0].(*widget.Check); ok {
			return c
		}
	}
	c := widget.NewCheck("", nil)
	cell.Objects = []fyne.CanvasObject{c}
	return c
}

func (rm *rcloneManager) cellLabel(cell *fyne.Container) *widget.Label {
	if len(cell.Objects) == 1 {
		if l, ok := cell.Objects[0].(*widget.Label); ok {
			return l
		}
	}
	l := widget.NewLabel("")
	cell.Objects = []fyne.CanvasObject{l}
	return l
}

func (rm *rcloneManager) cellActionButtons(cell *fyne.Container) (toggle, edit, del *widget.Button) {
	if len(cell.Objects) == 1 {
		if row, ok := cell.Objects[0].(*fyne.Container); ok && len(row.Objects) == 3 {
			if toggleWrap, ok := row.Objects[0].(*fyne.Container); ok && len(toggleWrap.Objects) == 1 {
				if t, ok := toggleWrap.Objects[0].(*widget.Button); ok {
					if e, ok := row.Objects[1].(*widget.Button); ok {
						if d, ok := row.Objects[2].(*widget.Button); ok {
							return t, e, d
						}
					}
				}
			}
		}
	}
	toggle = widget.NewButton("", nil)
	edit = widget.NewButton("편집", nil)
	del = widget.NewButton("삭제", nil)
	// "마운트" vs "해제" are different lengths, so without a fixed size the
	// button (and everything after it) shifts width every time it toggles.
	toggleFixed := container.New(layout.NewGridWrapLayout(fyne.NewSize(64, 34)), toggle)
	cell.Objects = []fyne.CanvasObject{container.NewHBox(toggleFixed, edit, del)}
	return toggle, edit, del
}

// displayDrive is the pure formatting rule shared by the table's 드라이브
// column: "" reads as "자동" (rclone picks an unused drive letter itself).
func displayDrive(drive string) string {
	drive = strings.TrimSpace(drive)
	if drive == "" {
		return "(자동)"
	}
	return drive
}

func statusLabel(running bool) string {
	if running {
		return "연결됨"
	}
	return "해제됨"
}

func toggleLabel(running bool) string {
	if running {
		return "해제"
	}
	return "마운트"
}
