package tetraterm

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type multipageModal struct {
	Display *Display
	Name    string
	*tview.Pages
}

func newMultipageModal(display *Display, name string, pageTexts ...string) *multipageModal {

	mp := &multipageModal{
		Display: display,
		Name:    name,
		Pages:   tview.NewPages(),
	}

	for i, page := range pageTexts {

		pageNum := i

		modal := tview.NewModal()

		modal.SetText(page)
		if pageNum > 0 {
			modal.AddButtons([]string{"Prev Page"})
		}
		modal.AddButtons([]string{"Close"})
		if pageNum < len(pageTexts)-1 {
			modal.AddButtons([]string{"Next Page"})
		}

		pageName := "Page " + strconv.Itoa(pageNum)

		page := mp.AddPage(pageName, modal, true, false)

		modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {

			_, frontPage := page.GetFrontPage()

			if modal == frontPage {

				if pageNum > 0 && buttonLabel == "Prev Page" {
					mp.Pages.SwitchToPage("Page " + strconv.Itoa(pageNum-1))
				} else if pageNum < len(pageTexts)-1 && buttonLabel == "Next Page" {
					mp.Pages.SwitchToPage("Page " + strconv.Itoa(pageNum+1))
				} else if buttonLabel == "Close" {
					// page, _ := mp.GetFrontPage()
					// mp.HidePage(page)
					mp.Display.Root.HidePage(mp.Name)
				}

			}

		})

		modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

			if event.Key() == tcell.KeyEscape {
				mp.Display.Root.HidePage(mp.Name)
				return nil
			}

			return event

		})

		if i == 0 {
			mp.SwitchToPage(pageName)
		}

	}

	return mp

}

func combineStringsWithSpaces(text ...string) string {
	return strings.Join(text, " ")
}

type Scrollbar struct {
	*tview.TextArea
	treeview   *tview.TreeView
	nodeScroll float64
	scrolling  bool
}

func NewScrollbar(treeview *tview.TreeView) *Scrollbar {

	sb := &Scrollbar{
		TextArea: tview.NewTextArea(),
		treeview: treeview,
	}

	sb.SetText("■■", false)
	sb.SetSelectedStyle(tcell.Style{}.Background(tcell.ColorBlack))
	sb.SetDrawFunc(sb.handleDrawing)
	sb.SetBorder(true)
	sb.SetBorderColor(tcell.ColorGreen)

	return sb

}

func (sb *Scrollbar) HandleMouseInput(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {

	x, y := event.Position()
	px, py, pw, ph := sb.GetRect()

	if action == tview.MouseLeftDown {
		if x >= px && x <= px+pw && y >= py && y <= py+ph {
			sb.scrolling = true
			sb.SetBorderColor(tcell.ColorAntiqueWhite)
			return tview.MouseMove, nil // Intercept input
		}
	} else if action == tview.MouseLeftUp {
		sb.scrolling = false
		sb.SetBorderColor(tcell.ColorGreen)
	}

	sb.nodeScroll = float64(y-1) / float64(ph-2) // The -1 gives it a little extra margin to make it easier to go to the top

	if sb.nodeScroll < 0 {
		sb.nodeScroll = 0
	}
	if sb.nodeScroll > 1 {
		sb.nodeScroll = 1
	}

	return action, event

}

func (sb *Scrollbar) handleDrawing(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {

	if sb.scrolling {
		rowCount := sb.treeview.GetRowCount()

		t := ""
		for i := 0; i < int(sb.nodeScroll*float64(height-1)); i++ {
			t += "\n"
		}
		sb.SetText(t+"■■", false)

		target := int(sb.nodeScroll * float64(rowCount))

		res := sb.ChildWithIndexInTree(target)

		if res != nil {
			sb.treeview.SetCurrentNode(res)
		}

	}

	return x, y, width, height

}

func (sb *Scrollbar) ScrollTo(index int) {

	rowCount := sb.treeview.GetRowCount()

	sb.nodeScroll = float64(index) / float64(rowCount)

	_, _, _, height := sb.GetRect()

	t := ""

	for i := 0; i < int(sb.nodeScroll*float64(height)); i++ {
		t += "\n"
	}
	sb.SetText(t+"■■", false)

}

func (sb *Scrollbar) ChildWithIndexInTree(targetIndex int) *tview.TreeNode {

	current := 0

	if targetIndex == current {
		return sb.treeview.GetRoot()
	}

	var lastOption *tview.TreeNode

	var loop func(targetNode *tview.TreeNode) *tview.TreeNode

	loop = func(targetNode *tview.TreeNode) *tview.TreeNode {

		if current == targetIndex {
			return targetNode
		}

		lastOption = targetNode

		current++

		children := targetNode.GetChildren()
		if len(children) > 0 && targetNode.IsExpanded() {
			for _, child := range children {
				if res := loop(child); res != nil {
					return res
				}
			}
		}

		return nil

	}

	if res := loop(sb.treeview.GetRoot()); res != nil {
		return res
	}

	return lastOption

}

func (sb *Scrollbar) ChildIndexInTree(treeNode *tview.TreeNode) int {

	current := 0

	if treeNode == sb.treeview.GetRoot() {
		return 0
	}

	var loop func(currentNode *tview.TreeNode) bool

	loop = func(currentNode *tview.TreeNode) bool {

		if currentNode == treeNode {
			return true
		}

		current++

		children := currentNode.GetChildren()
		if len(children) > 0 && currentNode.IsExpanded() {
			for _, child := range children {
				if res := loop(child); res {
					return res
				}
			}
		}

		return false

	}

	if res := loop(sb.treeview.GetRoot()); res {
		return current
	}

	return -1

}
