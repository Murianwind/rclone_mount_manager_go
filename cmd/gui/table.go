package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// table column indices — keep in sync with buildTable's header labels.
const (
	colKind = iota
	colAuto
	colDrive
	colRemote
	colStatus
	colActions
	colCount
)

func (rm *rcloneManager) buildTable() {
	rm.table = widget.NewTable(
		func() (int, int) { return len(rm.rows()), colCount },
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
		headers := [colCount]string{"구분", "자동", "드라이브", "리모트(서브경로)", "상태", ""}
		o.(*widget.Label).SetText(headers[id.Col])
	}
	rm.table.SetColumnWidth(colKind, 90)
	rm.table.SetColumnWidth(colAuto, 50)
	rm.table.SetColumnWidth(colDrive, 80)
	rm.table.SetColumnWidth(colRemote, 260)
	rm.table.SetColumnWidth(colStatus, 80)
	rm.table.SetColumnWidth(colActions, 200)

	rm.table.OnSelected = func(id widget.TableCellID) { rm.selectedRow = id.Row }
}

// updateTableCell fills in one cell. CreateCell can't know in advance
// which column a recycled template will be asked to render, so each
// helper (cellCheck/cellLabel/cellActionButtons) replaces the cell's
// content if it isn't already the right widget type.
func (rm *rcloneManager) updateTableCell(id widget.TableCellID, cell *fyne.Container) {
	rows := rm.rows()
	if id.Row >= len(rows) {
		return
	}
	row := rows[id.Row]

	if id.Col == colKind {
		rm.cellLabel(cell).SetText(kindLabel(row.kind))
		return
	}

	if row.kind == rowKindRemote {
		rm.updateRemoteRowCell(id.Col, cell, row.remote)
		return
	}
	rm.updateMountRowCell(id.Col, cell, row.mount)
}

func (rm *rcloneManager) updateRemoteRowCell(col int, cell *fyne.Container, r engine.Remote) {
	switch col {
	case colAuto, colDrive, colStatus:
		rm.cellLabel(cell).SetText("")
	case colRemote:
		rm.cellLabel(cell).SetText(remoteDisplayText(r))
	case colActions:
		importBtn, middleBtn, delBtn := rm.cellActionButtons(cell)
		importBtn.SetText("가져오기")
		importBtn.OnTapped = func() { rm.showMountDialog(nil, r.Name) }
		middleBtn.Hide() // 원본 행에는 편집 개념이 없어서 중간 슬롯을 안 씀
		delBtn.SetText("삭제")
		delBtn.OnTapped = func() { rm.confirmDeleteRemote(r) }
	}
}

func (rm *rcloneManager) updateMountRowCell(col int, cell *fyne.Container, m engine.Mount) {
	switch col {
	case colAuto:
		check := rm.cellCheck(cell)
		check.SetChecked(m.AutoMount)
		check.OnChanged = func(checked bool) {
			m.AutoMount = checked
			rm.saveMount(m)
		}
	case colDrive:
		rm.cellLabel(cell).SetText(displayDrive(m.Drive))
	case colRemote:
		rm.cellLabel(cell).SetText(fmt.Sprintf("%s:%s", m.Remote, m.RemotePath))
	case colStatus:
		rm.cellLabel(cell).SetText(statusLabel(rm.isRunning(m.ID)))
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
		editBtn.Show()
		editBtn.SetText("편집")
		editBtn.OnTapped = func() { rm.showMountDialog(&m, "") }
		delBtn.SetText("삭제")
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

// cellActionButtons returns a 3-button slot shared by both row kinds:
// mount rows use all three (토글/편집/삭제); remote rows only use the
// first and third (가져오기/삭제) — the middle one is left blank rather
// than removed, so the recycled widget shape stays consistent.
func (rm *rcloneManager) cellActionButtons(cell *fyne.Container) (first, middle, last *widget.Button) {
	if len(cell.Objects) == 1 {
		if row, ok := cell.Objects[0].(*fyne.Container); ok && len(row.Objects) == 3 {
			if firstWrap, ok := row.Objects[0].(*fyne.Container); ok && len(firstWrap.Objects) == 1 {
				if f, ok := firstWrap.Objects[0].(*widget.Button); ok {
					if m, ok := row.Objects[1].(*widget.Button); ok {
						if l, ok := row.Objects[2].(*widget.Button); ok {
							m.SetText("")
							m.OnTapped = nil
							return f, m, l
						}
					}
				}
			}
		}
	}
	first = widget.NewButton("", nil)
	middle = widget.NewButton("", nil)
	last = widget.NewButton("", nil)
	// 버튼 라벨 길이가 바뀌어도(마운트/해제/가져오기 등) 폭이 흔들리지
	// 않도록 첫 번째 버튼만 고정 폭으로 감싼다 — 뒤따르는 버튼들의
	// 위치가 흔들리는 걸 막기 위함.
	firstFixed := container.New(layout.NewGridWrapLayout(fyne.NewSize(64, 34)), first)
	cell.Objects = []fyne.CanvasObject{container.NewHBox(firstFixed, middle, last)}
	return first, middle, last
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
