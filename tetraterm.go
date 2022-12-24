package tetraterm

// TetraTerm is a terminal-based debugging solution for Tetra3D games and applications.

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	p2p "github.com/leprosus/golang-p2p"
	"github.com/rivo/tview"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"
)

type ogLocalTransform struct {
	Node     tetra3d.INode
	Parent   tetra3d.INode
	Position tetra3d.Vector
	Scale    tetra3d.Vector
	Rotation tetra3d.Matrix4
}

func newOGTransform(node tetra3d.INode) ogLocalTransform {
	return ogLocalTransform{
		Node:     node,
		Parent:   node.Parent(),
		Position: node.LocalPosition(),
		Scale:    node.LocalScale(),
		Rotation: node.LocalRotation(),
	}
}

func (transform ogLocalTransform) Apply(parentingAsWell bool) {

	if parentingAsWell && transform.Parent != transform.Node.Parent() {

		transform.Node.Unparent()

		if transform.Parent != nil {
			transform.Parent.AddChildren(transform.Node)
		}

	}

	transform.Node.SetLocalPositionVec(transform.Position)
	transform.Node.SetLocalScaleVec(transform.Scale)
	transform.Node.SetLocalRotation(transform.Rotation)

}

type emptyLogger struct{}

func (el emptyLogger) Info(msg string)  {}
func (el emptyLogger) Warn(msg string)  {}
func (el emptyLogger) Error(msg string) {}

type SceneNode struct {
	Name     string
	NodeID   uint64
	Children []SceneNode

	// Extra Stuff
	parent *SceneNode
}

func (st SceneNode) ChildrenRecursive() []SceneNode {
	out := []SceneNode{st}
	for _, child := range st.Children {
		out = append(out, child.ChildrenRecursive()...)
	}
	return out
}

func (st *SceneNode) Count() int {
	count := 1
	for _, child := range st.Children {
		count += child.Count()
	}
	return count
}

func (st *SceneNode) ResetParenting() {
	for c := range st.Children {
		st.Children[c].parent = st
		st.Children[c].ResetParenting()
	}
}

func constructNodeTree(t3dNode tetra3d.INode) SceneNode {

	sceneNode := SceneNode{
		Name:     t3dNode.Name(),
		NodeID:   t3dNode.ID(),
		Children: []SceneNode{},
	}

	for _, child := range t3dNode.Children() {
		newNode := constructNodeTree(child)
		sceneNode.Children = append(sceneNode.Children, newNode)
	}

	return sceneNode

}

type ConnectionSettings struct {
	Host          string
	Port          string
	SilentLogging bool
}

func NewDefaultConnectionSettings() *ConnectionSettings {
	return &ConnectionSettings{
		Host:          "", // Local host
		Port:          "7979",
		SilentLogging: true,
	}
}

// Server represents the endpoint of your game that the debugging Terminal hooks into.
// Note that this handles and records various settings to enable debugging simply and easily,
// so it's best to not instantiate a Server when shipping a release version of your game.
type Server struct {
	p2pServer    *p2p.Server
	Library      *tetra3d.Library
	activeScene  *tetra3d.Scene
	ogTransforms map[tetra3d.INode]ogLocalTransform
	prevScene    *tetra3d.Scene
	t3dCamera    *tetra3d.Camera
	selectedNode tetra3d.INode
}

// NewServer returns a new server, listening on the specified port number string (like "8000").
// You don't need to provide a library, but if you do, you'll be able to clone objects from other
// scenes from that library.
func NewServer(settings *ConnectionSettings, scene *tetra3d.Scene) *Server {

	if settings == nil {
		settings = NewDefaultConnectionSettings()
	}

	server := &Server{
		Library:      scene.Library(),
		selectedNode: scene.Root,
	}

	port := p2p.NewTCP(settings.Host, settings.Port)

	s, err := p2p.NewServer(port)
	if err != nil {
		panic(err)
	}

	if settings.SilentLogging {
		s.SetLogger(emptyLogger{})
	}

	server.p2pServer = s

	s.SetHandle(PTNodeFollowCamera, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		if server.selectedNode != server.t3dCamera {
			server.selectedNode.AddChildren(server.t3dCamera)
			server.t3dCamera.ResetLocalTransform()
			server.t3dCamera.Move(0, 0, 10)
		}

		return

	})

	s.SetHandle(PTNodeSelect, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeSelectPacket{}
		err = packet.Decode(req)
		if err != nil {
			panic(err)
		}

		nodes := append([]tetra3d.INode{server.activeScene.Root}, server.activeScene.Root.ChildrenRecursive()...)

		for _, node := range nodes {

			if node.ID() == packet.NodeID {
				server.selectedNode = node
				break
			}

		}

		return

	})

	s.SetHandle(PTNodeMove, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeMovePacket{}
		packet.Decode(req)
		server.selectedNode.Move(packet.X, packet.Y, packet.Z)
		return

	})

	s.SetHandle(PTNodeRotate, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeRotatePacket{}
		packet.Decode(req)
		server.selectedNode.Rotate(packet.X, packet.Y, packet.Z, packet.Angle)
		return

	})

	s.SetHandle(PTNodeReset, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeResetPacket{}
		packet.Decode(req)
		server.resetSelectedNode()
		return

	})

	s.SetHandle(PTNodeInfo, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeInfoPacket{}
		packet.Decode(req)
		packet.Position = server.selectedNode.LocalPosition()
		packet.Scale = server.selectedNode.LocalScale()
		packet.Rotation = server.selectedNode.LocalRotation().ToQuaternion()
		res = packet.Encode()
		return

	})

	s.SetHandle(PTNodeCreate, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeCreatePacket{}
		packet.Decode(req)

		if packet.NodeToCreate == "" {

			if server.Library != nil {

				nodeNames := []string{}

				for _, s := range server.Library.Scenes {
					for _, child := range s.Root.ChildrenRecursive() {
						candidate := child.Name()
						exists := false
						for _, n := range nodeNames {
							if candidate == n {
								exists = true
								break
							}
						}
						if !exists {
							nodeNames = append(nodeNames, candidate)
						}
					}
				}

				packet.ViableNodes = nodeNames

			}

		} else {

			var scenesToSearch []*tetra3d.Scene

			if server.Library != nil {
				scenesToSearch = server.Library.Scenes
			} else {
				scenesToSearch = []*tetra3d.Scene{server.activeScene}
			}

			for _, scene := range scenesToSearch {
				for _, node := range scene.Root.ChildrenRecursive() {
					if node.Name() == packet.NodeToCreate {
						clone := node.Clone()
						server.activeScene.Root.AddChildren(clone)
						packet.NewSelectedNode = clone.ID()
						server.selectedNode = clone
						packet.SceneTree = constructNodeTree(server.activeScene.Root)
					}
				}
			}

		}

		res = packet.Encode()

		return

	})

	s.SetHandle(PTNodeDuplicate, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeDuplicatePacket{}
		packet.Decode(req)

		if server.selectedNode != server.activeScene.Root {
			clone := server.selectedNode.Clone()

			replaceRandomNumber, _ := regexp.Compile("<[0123456789]{4}>")
			name := clone.Name()
			if match := replaceRandomNumber.FindStringIndex(name); match != nil {
				name = name[:match[0]]
			}
			clone.SetName(name + "<" + strconv.Itoa(rand.Intn(9999)) + ">")

			server.selectedNode.Parent().AddChildren(clone)
			server.selectedNode.Parent().ReindexChild(clone, server.selectedNode.Index()+1)
			packet.NewSelectedNode = clone.ID()
			server.selectedNode = clone
			packet.SceneTree = constructNodeTree(server.activeScene.Root)
		}

		res = packet.Encode()

		return

	})

	s.SetHandle(PTNodeDelete, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeDeletePacket{}
		packet.Decode(req)
		if server.selectedNode != server.activeScene.Root {

			ogIndex := server.selectedNode.Index()
			parent := server.selectedNode.Parent()

			server.selectedNode.Unparent()

			var newSelection tetra3d.INode

			if len(parent.Children()) > 0 {
				newSelection = parent.Children()[ogIndex-1]
			} else {
				newSelection = parent
			}

			packet.NewSelectedNode = newSelection.ID()
			server.selectedNode = newSelection
			packet.SceneTree = constructNodeTree(server.activeScene.Root)

		}

		res = packet.Encode()

		return

	})

	s.SetHandle(PTNodeMoveInTree, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &NodeMoveInTreePacket{}
		packet.Decode(req)

		if server.selectedNode != server.activeScene.Root {

			node := server.selectedNode
			switch packet.MoveDir {
			case MITMoveUp:
				node.Parent().ReindexChild(node, node.Index()-1)
			case MITMoveDown:
				node.Parent().ReindexChild(node, node.Index()+1)
			case MITIndent:
				if node.Index() > 0 {
					node.Parent().Children()[node.Index()-1].AddChildren(node)
				}
			case MITDeIndent:
				if node.Parent() != node.Root() {
					ogParentIndex := node.Parent().Index()
					node.Parent().Parent().AddChildren(node)
					node.Parent().ReindexChild(node, ogParentIndex+1)
				}
			}

			packet.NewSelectedNode = node.ID()
			packet.SceneTree = constructNodeTree(server.activeScene.Root)

		}

		res = packet.Encode()

		return

	})

	s.SetHandle(PTSceneRefresh, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &SceneRefreshPacket{}
		err = packet.Decode(req)

		if err != nil {
			log.Println(err)
		} else {

			packetChanged := false

			if server.activeScene != nil {
				tree := constructNodeTree(server.activeScene.Root)
				packet.SceneTree = tree
				packetChanged = true
			} else {
				log.Println("warning: no scene set for server")
			}

			// Give the response
			if packetChanged {
				res = packet.Encode()
			}

		}

		return
	})

	go func() {
		if err := server.p2pServer.Serve(); err != nil {
			panic(err)
		}
	}()

	return server

}

// Update updates the server as necessary.
func (server *Server) Update(scene *tetra3d.Scene) {
	server.activeScene = scene
	server.Library = scene.Library()

	if server.selectedNode == nil {
		server.selectedNode = server.activeScene.Root
	}

	if server.activeScene != server.prevScene {
		server.ogTransforms = map[tetra3d.INode]ogLocalTransform{}
	}

	nodes := append([]tetra3d.INode{scene.Root}, scene.Root.ChildrenRecursive()...)

	for _, n := range nodes {
		server.recordOGTransforms(n)
	}

	server.prevScene = server.activeScene

}

// Draw handles any additional drawing from the terminal.
func (server *Server) Draw(screen *ebiten.Image, camera *tetra3d.Camera) {

	server.t3dCamera = camera

	camera.DrawDebugCenters(screen, server.selectedNode, colors.White())

	draw := func(node tetra3d.INode) {
		if node != camera {
			pos := camera.WorldToScreen(node.WorldPosition())
			color := colors.Gray()
			if node == server.selectedNode {
				color = colors.White()
			}
			camera.DebugDrawText(screen, node.Name(), pos.X, pos.Y, 1, color)
		}
	}

	draw(server.selectedNode)

	for _, n := range server.selectedNode.ChildrenRecursive() {
		draw(n)
	}

}

func (server *Server) recordOGTransforms(node tetra3d.INode) {

	if _, exists := server.ogTransforms[node]; !exists {
		server.ogTransforms[node] = newOGTransform(node)
	}

}

func (server *Server) resetSelectedNode() {

	server.ogTransforms[server.selectedNode].Apply(true)

	for _, node := range server.selectedNode.ChildrenRecursive() {
		server.ogTransforms[node].Apply(false)
	}

}

// Display represents the terminal interface that hooks into your game through a TCP connection (default being on port 8000).
// By manipulating the Display, you can view and modify your game scene while it's running. It will also hook into whatever
// application is listening on the connected port, so you can leave it running while you stop and recompile your game; it will
// update to match.
type Display struct {
	Client         *p2p.Client
	ClientSettings *ConnectionSettings

	App *tview.Application

	Root *tview.Pages

	running          atomic.Bool
	currentSceneTree SceneNode

	receivingData atomic.Bool

	// Flexbox *tview.Flex

	TreeNodeRoot *tview.TreeNode
	TreeView     *tview.TreeView

	propertyArea *tview.TextArea

	searchBar                           *tview.InputField
	searchBarCloneMode                  bool
	searchBarCloneModeAutocompleteNames []string

	// prevSceneData map[tetra3d.INode]string

	// propertyText *tview.TextArea

	selectNextNode      bool
	selectNextNodeIndex uint64

	sceneNodesToTreeNodes map[uint64]*tview.TreeNode
	// DebugDraw    bool

	// deleteNode        bool
	// duplicateNode     bool
	// stopped           atomic.Bool
	// forceSceneRebuild bool

	// Handlers

	// Scene  *tetra3d.Scene
	// Camera *tetra3d.Camera
}

func NewDisplay(settings *ConnectionSettings) *Display {

	if settings == nil {
		settings = NewDefaultConnectionSettings()
	}

	app := &Display{

		ClientSettings: settings,

		App:     tview.NewApplication(),
		running: atomic.Bool{},

		sceneNodesToTreeNodes: map[uint64]*tview.TreeNode{},

		// Flexbox: tview.NewFlex(),

		// prevSceneData: map[tetra3d.INode]string{},

		// stopped:      atomic.Bool{},
		// ogTransforms: map[tetra3d.INode]ogLocalTransform{},
	}

	app.initClient()

	/////////

	app.Root = tview.NewPages()
	app.App.SetRoot(app.Root, true)

	app.TreeView = tview.NewTreeView()
	app.TreeView.SetBorder(true)
	app.TreeView.SetTitle("[ ◆ Node Tree ]")
	app.TreeView.SetGraphicsColor(tcell.ColorGreen)

	app.TreeView.SetSelectedFunc(func(node *tview.TreeNode) {

		if len(node.GetChildren()) > 0 {
			node.SetExpanded(!node.IsExpanded())
		}

		app.updateTreeNodeNames()

	})

	leftSide := tview.NewFlex()
	leftSide.SetDirection(tview.FlexRow)

	treeFlex := tview.NewFlex()
	treeFlex.AddItem(leftSide, 0, 2, true)

	leftSide.AddItem(app.TreeView, 0, 2, true)

	app.searchBar = tview.NewInputField()
	app.searchBar.SetAutocompleteFunc(func(currentText string) (entries []string) {

		if currentText == "" {
			return
		}

		var nodeNames []string

		if app.searchBarCloneMode {
			nodeNames = app.searchBarCloneModeAutocompleteNames
		} else {

			nodeNames = []string{app.currentSceneTree.Name}

			for _, n := range app.currentSceneTree.ChildrenRecursive() {
				nodeNames = append(nodeNames, n.Name)
			}

		}

		for _, n := range nodeNames {
			if strings.Contains(strings.ToLower(n), strings.ToLower(currentText)) {
				entries = append(entries, n)
			}
		}

		return

	})
	app.searchBar.SetLabel("Search Node: ")
	app.searchBar.SetLabelColor(tcell.ColorLightBlue)
	app.searchBar.SetFieldBackgroundColor(tcell.ColorDarkSlateBlue)
	app.searchBar.SetFieldTextColor(tcell.ColorLightBlue)

	app.searchBar.SetChangedFunc(func(text string) {

		if app.searchBarCloneMode {
			res, err := app.sendRequest(NewNodeCreatePacket())
			if err != nil {
				log.Println(err)
			} else if res.(*NodeCreatePacket).ViableNodes != nil {
				app.searchBarCloneModeAutocompleteNames = res.(*NodeCreatePacket).ViableNodes
			}
		}

		// Hacky fix
		app.TreeView.GetRoot().ExpandAll()

		if text != "" {

			nodesToSearch := append([]SceneNode{app.currentSceneTree}, app.currentSceneTree.ChildrenRecursive()...)

			for _, node := range nodesToSearch {

				if strings.Contains(strings.ToLower(node.Name), strings.ToLower(text)) {
					treeNode := app.sceneNodesToTreeNodes[node.NodeID]
					treeNode.ExpandAll()
					app.TreeView.SetCurrentNode(treeNode)
					app.sendRequest(NewNodeSelectPacket(node.NodeID))
					break
				}

			}

		}

	})

	// TODO: Be able to search for node by type or tag?

	// TODO: Pressing Shift+F while the search box is highlit should search for the next occurance
	// of the name
	// app.searchBar.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	if event.Rune() == 'F' {
	// 	}
	// })

	app.searchBar.SetDoneFunc(func(key tcell.Key) {
		if app.searchBarCloneMode && key == tcell.KeyEnter {
			np := NewNodeCreatePacket()
			np.NodeToCreate = app.searchBar.GetText()
			res, err := app.sendRequest(np)
			if err != nil {
				log.Println(err)
			} else {
				app.selectNextNode = true
				app.currentSceneTree = res.(*NodeCreatePacket).SceneTree
				app.selectNextNodeIndex = res.(*NodeCreatePacket).NewSelectedNode
			}
		}
		app.App.SetFocus(app.TreeView)
	})

	leftSide.AddItem(app.searchBar, 1, 0, false)

	//

	helpText := `Arrow Keys: Select Node
wasd, qe: Move Node in-game
r: Reset Node Position / Parent

f: Follow Node with Camera
Shift+F: Search

Shift+Arrows: Change Index / Parenting

Shift+D: Duplicate Node
Shift+X: Delete Node

F2: Search
	`

	rightSide := tview.NewFlex()
	rightSide.SetDirection(tview.FlexRow)
	treeFlex.AddItem(rightSide, 0, 1, false)

	t := tview.NewTextArea().SetText(helpText, false)
	t.SetBorder(true).SetTitle("[ Keys ]")
	rightSide.AddItem(t, 0, 1, false)
	// treeFlex.SetDirection(tview.FlexRow)

	app.propertyArea = tview.NewTextArea()
	app.propertyArea.SetBorder(true)
	app.propertyArea.SetTitle("[ Properties ]")
	rightSide.AddItem(app.propertyArea, 0, 1, false)

	app.Root.AddAndSwitchToPage("Tree View", treeFlex, true)

	// debugText := tview.NewTextArea()

	// // app.App.SetAfterDrawFunc(func(screen tcell.Screen) {
	// // 	// log.Println(resp)
	// // })

	go func() {

		for {

			time.Sleep(time.Millisecond * 250)
			app.receivingData.Store(true)

			resp, err := app.sendRequest(NewSceneRefreshPacket())

			if err != nil {
				if err.Error() == "EOF" {
					// Attempt to reopen connection
					// log.Println("Request failed, attempting to reopen connection:")
					app.initClient()
					app.App.Draw()
				} else {
					panic(err)
				}
			} else {
				app.currentSceneTree = resp.(*SceneRefreshPacket).SceneTree

				app.currentSceneTree.ResetParenting()

				for _, node := range app.currentSceneTree.ChildrenRecursive() {

					existingNode, exists := app.sceneNodesToTreeNodes[node.NodeID]

					if !exists {
						tn := tview.NewTreeNode(node.Name)
						tn.SetSelectable(true)
						tn.SetReference(node)
						app.sceneNodesToTreeNodes[node.NodeID] = tn
						existingNode = tn
					}

					existingNode.ClearChildren()

					if node.parent != nil {
						app.sceneNodesToTreeNodes[node.parent.NodeID].AddChild(existingNode)
					}

				}

				app.TreeNodeRoot = app.sceneNodesToTreeNodes[app.currentSceneTree.NodeID]
				app.TreeNodeRoot.SetSelectable(true)
				app.TreeNodeRoot.SetColor(tcell.ColorSkyblue)
				app.TreeView.SetRoot(app.TreeNodeRoot)
				if app.TreeView.GetCurrentNode() == nil {
					app.TreeView.SetCurrentNode(app.TreeNodeRoot)
				}

				if app.selectNextNode {
					app.TreeView.SetCurrentNode(app.sceneNodesToTreeNodes[app.selectNextNodeIndex])
					app.selectNextNode = false
				}

				app.updateTreeNodeNames()

				app.App.Draw()

			}

			app.receivingData.Store(false)
		}

	}()

	go func() {
		for {
			time.Sleep(time.Millisecond * 100)

			app.TreeView.SetTitle(fmt.Sprintf("[ ◆ Node Tree ] x [%d Nodes]", app.currentSceneTree.Count()))

			resp, err := app.sendRequest(NewNodeInfoPacket())
			if err != nil {
				log.Println(err)
			} else {
				info := resp.(*NodeInfoPacket)
				text := fmt.Sprintf("Pos:%v\nSca:%v\nRot:%v", info.Position, info.Scale, info.Rotation)
				app.App.QueueUpdate(func() {
					app.propertyArea.SetText(text, false)
				})
				// fmt.Println(info)
			}

			// app.receivingData.Store(true)
			// app.receivingData.Store(false)
		}
	}()

	app.TreeView.SetChangedFunc(func(node *tview.TreeNode) {
		app.sendRequest(NewNodeSelectPacket(node.GetReference().(SceneNode).NodeID))
	})

	app.TreeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// tvNode := app.TreeView.GetCurrentNode()

		// sceneNode := tvNode.GetReference().(SceneNode)

		// app.recordOriginalSettings(t3dNode.Scene().Root)

		// for _, node := range t3dNode.Scene().Root.ChildrenRecursive() {
		// 	app.recordOriginalSettings(node)
		// }

		if event.Rune() == 'f' {
			app.sendRequest(NewNodeFollowCameraPacket())
			return nil
		}

		if event.Rune() == 'F' {
			app.searchBar.SetText("")
			app.searchBar.SetLabel("Search Node: ")
			app.searchBarCloneMode = false
			app.App.SetFocus(app.searchBar)
			return nil
		}

		if event.Rune() == 'C' {
			app.searchBar.SetText("")
			app.searchBar.SetLabel("Clone Node: ")
			app.searchBarCloneMode = true
			app.App.SetFocus(app.searchBar)
			return nil
		}

		if event.Rune() == 'D' {
			app.receivingData.Store(true)
			res, err := app.sendRequest(NewNodeDuplicatePacket())
			if err != nil {
				log.Println(err)
			} else {
				app.selectNextNode = true
				app.currentSceneTree = res.(*NodeDuplicatePacket).SceneTree
				app.selectNextNodeIndex = res.(*NodeDuplicatePacket).NewSelectedNode
			}
			app.receivingData.Store(false)
			return nil
		}
		if event.Rune() == 'X' {
			app.receivingData.Store(true)
			res, err := app.sendRequest(NewNodeDeletePacket())
			if err != nil {
				log.Println(err)
			} else {
				app.selectNextNode = true
				app.currentSceneTree = res.(*NodeDeletePacket).SceneTree
				app.selectNextNodeIndex = res.(*NodeDeletePacket).NewSelectedNode
			}
			app.receivingData.Store(false)
			return nil
		}
		// if event.Rune() == 'X' {
		// 	app.deleteNode = true
		// 	return nil
		// }

		// if t3dNode.Parent() != nil {

		if (event.Key() == tcell.KeyUp || event.Key() == tcell.KeyRight || event.Key() == tcell.KeyDown || event.Key() == tcell.KeyLeft) && event.Modifiers() == tcell.ModShift {

			var moveDir int

			switch event.Key() {
			case tcell.KeyUp:
				moveDir = MITMoveUp
			case tcell.KeyDown:
				moveDir = MITMoveDown
			case tcell.KeyRight:
				moveDir = MITIndent
			case tcell.KeyLeft:
				moveDir = MITDeIndent
			}

			packet := NewNodeMoveInTreePacket(moveDir)

			app.receivingData.Store(true)

			res, err := app.sendRequest(packet)
			if err != nil {
				log.Println(err)
			} else {
				app.selectNextNode = true
				app.currentSceneTree = res.(*NodeMoveInTreePacket).SceneTree
				app.selectNextNodeIndex = res.(*NodeMoveInTreePacket).NewSelectedNode
			}

			app.receivingData.Store(false)

			return nil

		}

		// }

		moveDist := 1.0
		if event.Rune() == 'w' {
			app.sendRequest(NewNodeMovePacket(0, 0, -moveDist))
			return nil
		}
		if event.Rune() == 'd' {
			app.sendRequest(NewNodeMovePacket(moveDist, 0, 0))
			return nil
		}

		if event.Rune() == 's' {
			app.sendRequest(NewNodeMovePacket(0, 0, moveDist))
			return nil
		}
		if event.Rune() == 'a' {
			app.sendRequest(NewNodeMovePacket(-moveDist, 0, 0))
			return nil
		}

		if event.Rune() == 'q' {
			app.sendRequest(NewNodeMovePacket(0, -moveDist, 0))
			return nil
		}

		if event.Rune() == 'e' {
			app.sendRequest(NewNodeMovePacket(0, moveDist, 0))
			return nil
		}

		// if event.Rune() == 'z' {
		// 	app.sendRequest(NewNodeRotatePacket(1, 0, 0, tetra3d.ToRadians(10)))
		// 	return nil
		// }
		// if event.Rune() == 'Z' {
		// 	app.sendRequest(NewNodeRotatePacket(1, 0, 0, -tetra3d.ToRadians(10)))
		// 	return nil
		// }

		// if event.Rune() == 'x' {
		// 	app.sendRequest(NewNodeRotatePacket(0, 1, 0, tetra3d.ToRadians(10)))
		// 	return nil
		// }
		// if event.Rune() == 'X' {
		// 	app.sendRequest(NewNodeRotatePacket(0, 1, 0, -tetra3d.ToRadians(10)))
		// 	return nil
		// }

		// if event.Rune() == 'c' {
		// 	app.sendRequest(NewNodeRotatePacket(0, 0, 1, tetra3d.ToRadians(10)))
		// 	return nil
		// }
		// if event.Rune() == 'C' {
		// 	app.sendRequest(NewNodeRotatePacket(0, 0, 1, -tetra3d.ToRadians(10)))
		// 	return nil
		// }

		// Reset Node
		if event.Rune() == 'r' {
			app.sendRequest(NewNodeResetPacket())
			return nil
		}

		return event
	})

	// 	app.TreeNodeRoot = tview.NewTreeNode("ROOT").SetColor(tcell.ColorLightSkyBlue)
	// 	app.TreeView = tview.NewTreeView().SetRoot(app.TreeNodeRoot).SetCurrentNode(app.TreeNodeRoot)
	// 	app.TreeView.SetTitle("[ ◆ Node Tree ]")
	// 	app.TreeView.Box.SetBorder(true)
	// 	app.TreeView.SetGraphicsColor(tcell.ColorGreen)

	// 	app.Flexbox.AddItem(app.TreeView, 0, 1, true)
	// 	helpText := `f: Focus Camera
	// Arrow Keys: Select Node
	// wasd, qe: Move Node
	// r: Reset Node
	// Shift+Arrows: Change Parenting

	// Shift+D: Duplicate Node
	// Shift+X: Delete Node

	// `
	// 	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	// 	app.Flexbox.AddItem(flex, 0, 1, false)

	// 	t := tview.NewTextArea().SetText(helpText, false)
	// 	t.SetBorder(true).SetTitle("[ Keys ]")
	// 	flex.AddItem(t, 0, 2, false)
	// 	app.App.SetRoot(app.Flexbox, true)

	// 	app.propertyText = tview.NewTextArea().SetText("Property text goes here", false)
	// 	app.propertyText.SetBorder(true).SetTitle("[ Properties ]")
	// 	flex.AddItem(app.propertyText, 0, 2, false)

	return app

}

func (td *Display) initClient() {

	port := p2p.NewTCP(td.ClientSettings.Host, td.ClientSettings.Port)

	client, err := p2p.NewClient(port)
	if err != nil {
		panic(err)
	}

	settings := p2p.NewClientSettings()
	settings.SetRetry(1000, time.Millisecond*500)
	client.SetSettings(settings)

	if td.ClientSettings.SilentLogging {
		client.SetLogger(emptyLogger{})
	}

	td.Client = client

}

func (td *Display) Start() error {

	td.running.Store(true)

	// go func() {
	// 	for {
	// 		if td.running.Load() {
	// 			td.App.Draw()
	// 		} else {
	// 			return
	// 		}
	// 	}
	// }()

	return td.App.EnableMouse(true).Run()

}

func (td Display) sendRequest(packet IPacket) (IPacket, error) {

	response, err := td.Client.Send(packet.DataType(), packet.Encode())
	if err != nil {
		return nil, err
	}

	return packet, packet.Decode(response)
	// return DecodePacket(packet.DataType(), response)

}

func (display *Display) updateTreeNodeNames() {

	for _, node := range display.currentSceneTree.ChildrenRecursive() {

		tn := display.sceneNodesToTreeNodes[node.NodeID]

		name := node.Name

		if len(node.Children) > 0 {
			if tn.IsExpanded() {
				name = "[=] " + name
			} else {
				name = "[+] " + name
			}
		} else {
			name = " +  " + name
		}

		tn.SetText(name)

	}

	// display.TreeNodeRoot.Walk(func(node, parent *tview.TreeNode) bool {

	// 	t3dNode := node.GetReference().(tetra3d.INode)

	// 	if len(node.GetChildren()) > 0 {

	// 		t := "[+] "
	// 		if node.IsExpanded() {
	// 			t = "[=] "
	// 		}
	// 		node.SetText(t + t3dNode.Name())

	// 	} else {
	// 		node.SetText(" +  " + t3dNode.Name())
	// 	}

	// 	return true
	// })

}

// func (app *TerminalApp) Recover() {
// 	if result := recover(); result != nil {
// 		app.App.Stop()
// 		panic(result)
// 	}
// }

// func (app *TerminalApp) Draw(screen *ebiten.Image, scene *tetra3d.Scene) {

// 	if !app.stopped.Load() {

// 		app.TreeNodeRoot.SetReference(scene.Root)

// 		sceneChanged := app.forceSceneRebuild
// 		app.forceSceneRebuild = false

// 		if len(scene.Root.ChildrenRecursive()) != len(app.prevSceneData) {
// 			sceneChanged = true
// 		} else {

// 			for _, n := range scene.Root.ChildrenRecursive() {
// 				if prevPath, ok := app.prevSceneData[n]; !ok || prevPath != n.Path() {
// 					sceneChanged = true
// 					break
// 				}
// 			}

// 		}

// 		app.App.QueueUpdateDraw(func() {

// 			if app.duplicateNode {
// 				t3dNode := app.TreeView.GetCurrentNode().GetReference().(tetra3d.INode)
// 				if t3dNode != scene.Root {
// 					clone := t3dNode.Clone()
// 					name := clone.Name()
// 					a, _ := regexp.Compile("<[0123456789]{4}>")
// 					if match := a.FindStringIndex(name); match != nil {
// 						fmt.Println(match)
// 						name = name[:match[0]]
// 					}
// 					clone.SetName(name + "<" + strconv.Itoa(rand.Intn(9999)) + ">")
// 					t3dNode.Parent().AddChildren(clone)

// 					tn := tview.NewTreeNode("")
// 					tn.SetSelectable(true)
// 					tn.SetReference(clone)
// 					app.nodesToTreeNodes[clone] = tn
// 					app.TreeView.SetCurrentNode(tn)

// 					app.duplicateNode = false
// 					sceneChanged = true
// 				}
// 			}

// 			if app.deleteNode {
// 				t3dNode := app.TreeView.GetCurrentNode().GetReference().(tetra3d.INode)
// 				if t3dNode != scene.Root {
// 					t3dNode.Unparent()
// 					app.deleteNode = false
// 					sceneChanged = true
// 				}
// 			}

// 			if sceneChanged {

// 				for _, node := range scene.Root.ChildrenRecursive() {
// 					if _, exists := app.nodesToTreeNodes[node]; !exists {
// 						tn := tview.NewTreeNode("")
// 						tn.SetSelectable(true)
// 						tn.SetReference(node)
// 						app.nodesToTreeNodes[node] = tn
// 					}
// 				}

// 				var setupTree func(node tetra3d.INode)

// 				setupTree = func(node tetra3d.INode) {

// 					var tn *tview.TreeNode

// 					if node == scene.Root {
// 						tn = app.TreeNodeRoot
// 					} else {
// 						tn = app.nodesToTreeNodes[node]
// 					}

// 					tn.ClearChildren()

// 					for _, child := range node.Children() {
// 						tn.AddChild(app.nodesToTreeNodes[child])
// 					}

// 					for _, child := range node.Children() {
// 						setupTree(child)
// 					}

// 				}

// 				setupTree(scene.Root)

// 				app.updateTreeNodeNames()

// 			}

// 		})

// 		if sceneChanged {
// 			newMap := map[tetra3d.INode]string{}
// 			for _, n := range scene.Root.ChildrenRecursive() {
// 				newMap[n] = n.Path()
// 			}
// 			app.prevSceneData = newMap
// 		}

// 		tvNode := app.TreeView.GetCurrentNode()
// 		if tvNode != nil {
// 			t3dNode := tvNode.GetReference().(tetra3d.INode)

// 			app.propertyText.SetText(
// 				fmt.Sprintf("Pos: %s\nSca: %s\nRot:\n%s", t3dNode.WorldPosition(), t3dNode.WorldScale(), app.quatToShortString(t3dNode.WorldRotation().ToQuaternion())),
// 				false,
// 			)

// 			app.Camera.DrawDebugCenters(screen, t3dNode, colors.White())

// 			drawName := func(node tetra3d.INode) {
// 				if node == app.Camera {
// 					return
// 				}
// 				np := app.Camera.WorldToScreen(node.WorldPosition())
// 				app.Camera.DebugDrawText(screen, node.Name(), np.X, np.Y, 1, colors.White())
// 			}

// 			drawName(t3dNode)

// 			for _, node := range t3dNode.ChildrenRecursive() {
// 				drawName(node)
// 			}
// 		}

// 	}

// }

// func (ta *TerminalApp) Stop() {
// 	ta.App.QueueUpdate(func() { ta.App.Stop() })
// }

// func (ta *TerminalApp) quatToShortString(quat tetra3d.Quaternion) string {
// 	s := "{"
// 	s += strconv.FormatFloat(quat.X, 'f', 2, 64) + ", "
// 	s += strconv.FormatFloat(quat.Y, 'f', 2, 64) + ", "
// 	s += strconv.FormatFloat(quat.Z, 'f', 2, 64) + ", "
// 	s += strconv.FormatFloat(quat.W, 'f', 2, 64) + ", "
// 	s += "}"
// 	return s
// }

// func (ta *TerminalApp) matrixToShortString(matrix tetra3d.Matrix4) string {
// 	s := "{"
// 	for i, y := range matrix {
// 		for _, x := range y {
// 			s += strconv.FormatFloat(x, 'f', 2, 64) + ", "
// 		}
// 		if i < len(matrix)-1 {
// 			s += "\n"
// 		}
// 	}
// 	s += "}"
// 	return s
// }
