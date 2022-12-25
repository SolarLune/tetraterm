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

type sceneNode struct {
	Name     string
	NodeID   uint64
	Children []sceneNode

	// Extra Stuff
	parent *sceneNode
}

func (st sceneNode) ChildrenRecursive() []sceneNode {
	out := []sceneNode{st}
	for _, child := range st.Children {
		out = append(out, child.ChildrenRecursive()...)
	}
	return out
}

func (st *sceneNode) Count() int {
	count := 1
	for _, child := range st.Children {
		count += child.Count()
	}
	return count
}

func (st *sceneNode) ResetParenting() {
	for c := range st.Children {
		st.Children[c].parent = st
		st.Children[c].ResetParenting()
	}
}

func constructNodeTree(t3dNode tetra3d.INode) sceneNode {

	sceneNode := sceneNode{
		Name:     t3dNode.Name(),
		NodeID:   t3dNode.ID(),
		Children: []sceneNode{},
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
	P2PServer     *p2p.Server
	activeLibrary *tetra3d.Library
	activeScene   *tetra3d.Scene
	ogTransforms  map[tetra3d.INode]ogLocalTransform
	prevScene     *tetra3d.Scene
	t3dCamera     *tetra3d.Camera
	selectedNode  tetra3d.INode

	DebugDraw bool
}

// NewServer returns a new server, listening on the specified port number string (like "8000").
// You don't need to provide a library, but if you do, you'll be able to clone objects from other
// scenes from that library.
func NewServer(settings *ConnectionSettings, scene *tetra3d.Scene) *Server {

	if settings == nil {
		settings = NewDefaultConnectionSettings()
	}

	server := &Server{
		activeLibrary: scene.Library(),
		selectedNode:  scene.Root,
	}

	port := p2p.NewTCP(settings.Host, settings.Port)

	s, err := p2p.NewServer(port)
	if err != nil {
		panic(err)
	}

	if settings.SilentLogging {
		s.SetLogger(emptyLogger{})
	}

	server.P2PServer = s

	s.SetHandle(ptNodeFollowCamera, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		if server.t3dCamera != nil {

			if server.selectedNode != server.t3dCamera {
				server.selectedNode.AddChildren(server.t3dCamera)
				server.t3dCamera.ResetLocalTransform()
				server.t3dCamera.Move(0, 0, 10)
			}

		}

		return

	})

	s.SetHandle(ptToggleDebugDraw, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		server.DebugDraw = !server.DebugDraw

		packet := toggleDebugDraw{}
		packet.Decode(req)
		packet.DebugDrawOn = server.DebugDraw
		res = packet.Encode()

		return

	})

	s.SetHandle(ptNodeSelect, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &nodeSelectPacket{}
		err = packet.Decode(req)
		if err != nil {
			panic(err)
		}

		if server.activeScene != nil {

			nodes := append([]tetra3d.INode{server.activeScene.Root}, server.activeScene.Root.ChildrenRecursive()...)

			for _, node := range nodes {

				if node.ID() == packet.NodeID {
					server.selectedNode = node
					break
				}

			}

		}

		return

	})

	s.SetHandle(ptNodeMove, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		if server.selectedNode != nil {

			packet := &nodeMovePacket{}
			packet.Decode(req)
			server.selectedNode.Move(packet.X, packet.Y, packet.Z)

		}

		return

	})

	s.SetHandle(ptNodeRotate, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		if server.selectedNode != nil {

			packet := &nodeRotatePacket{}
			packet.Decode(req)
			server.selectedNode.Rotate(packet.X, packet.Y, packet.Z, packet.Angle)

		}

		return

	})

	s.SetHandle(ptNodeReset, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		server.resetSelectedNode()
		return

	})

	s.SetHandle(ptNodeInfo, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		if server.selectedNode != nil {

			packet := &nodeInfoPacket{}
			packet.ID = server.selectedNode.ID()
			packet.Position = server.selectedNode.LocalPosition()
			packet.Scale = server.selectedNode.LocalScale()
			packet.Rotation = matrix4ToMatrix3(server.selectedNode.LocalRotation())
			packet.Visible = server.selectedNode.Visible()
			packet.Type = server.selectedNode.Type()
			res = packet.Encode()

		}

		return

	})

	s.SetHandle(ptGameInfo, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		// We don't need to decode this packet because the client is asking for this, and has nothing to
		// give us.
		packet := &gameInfoPacket{
			FPS: float32(ebiten.ActualFPS()),
			TPS: float32(ebiten.ActualTPS()),
			// ModelCount: server.activeScene.Root.ChildrenRecursive().ByType(tetra3d.NodeTypeModel).,
		}

		if server.t3dCamera != nil {

			packet.DebugInfo = server.t3dCamera.DebugInfo

		}

		res = packet.Encode()
		return

	})

	s.SetHandle(ptNodeCreate, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &nodeCreatePacket{}
		packet.Decode(req)

		if packet.NodeToCreate == "" {

			if server.activeLibrary != nil {

				nodeNames := []string{}

				for _, s := range server.activeLibrary.Scenes {
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
				res = packet.Encode()
				return
			}

		} else {

			var scenesToSearch []*tetra3d.Scene

			if server.activeLibrary != nil {
				scenesToSearch = server.activeLibrary.Scenes
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
						res = packet.Encode()
						return

					}

				}

			}

		}

		return

	})

	s.SetHandle(ptNodeDuplicate, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &nodeDuplicatePacket{}
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

	s.SetHandle(ptNodeDelete, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &nodeDeletePacket{}
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

	s.SetHandle(ptNodeMoveInTree, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &nodeMoveInTreePacket{}
		packet.Decode(req)

		if server.selectedNode != server.activeScene.Root {

			node := server.selectedNode
			switch packet.MoveDir {
			case mitMoveUp:
				node.Parent().ReindexChild(node, node.Index()-1)
			case mitMoveDown:
				node.Parent().ReindexChild(node, node.Index()+1)
			case mitIndent:
				if node.Index() > 0 {
					node.Parent().Children()[node.Index()-1].AddChildren(node)
				}
			case mitDeIndent:
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

	s.SetHandle(ptSceneRefresh, func(ctx context.Context, req p2p.Data) (res p2p.Data, err error) {

		packet := &sceneRefreshPacket{}
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
		if err := server.P2PServer.Serve(); err != nil {
			panic(err)
		}
	}()

	return server

}

// Update updates the server as necessary. This should be called every tick.
func (server *Server) Update(scene *tetra3d.Scene) {
	server.activeScene = scene
	server.activeLibrary = scene.Library()

	if server.selectedNode == nil {
		server.selectedNode = server.activeScene.Root
	}

	// Scene changed, so we can empty the og transforms list.
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

	if server.DebugDraw {

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

}

func (server *Server) recordOGTransforms(node tetra3d.INode) {

	if _, exists := server.ogTransforms[node]; !exists {
		server.ogTransforms[node] = newOGTransform(node)
	}

}

func (server *Server) resetSelectedNode() {

	if server.selectedNode != nil {

		server.ogTransforms[server.selectedNode].Apply(true)

		for _, node := range server.selectedNode.ChildrenRecursive() {
			server.ogTransforms[node].Apply(false)
		}

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
	currentSceneTree sceneNode

	receivingData atomic.Bool

	// Flexbox *tview.Flex

	TreeNodeRoot *tview.TreeNode
	TreeView     *tview.TreeView

	NodePropertyArea *tview.TextArea
	GamePropertyArea *tview.TextArea

	SearchBar                           *tview.InputField
	SearchBarCloneMode                  bool
	SearchBarCloneModeAutocompleteNames []string

	// prevSceneData map[tetra3d.INode]string

	// propertyText *tview.TextArea

	SelectNextNode      bool
	SelectNextNodeIndex uint64

	SceneNodesToTreeNodes map[uint64]*tview.TreeNode
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

		SceneNodesToTreeNodes: map[uint64]*tview.TreeNode{},

		// Flexbox: tview.NewFlex(),

		// prevSceneData: map[tetra3d.INode]string{},

		// stopped:      atomic.Bool{},
		// ogTransforms: map[tetra3d.INode]ogLocalTransform{},
	}

	app.initClient()

	/////////

	app.Root = tview.NewPages()
	app.Root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		if event.Rune() == '1' {
			app.sendRequest(newToggleDebugDraw())
		}

		if event.Key() == tcell.KeyCtrlQ {
			app.App.Stop()
		}

		return event

	})
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

	app.SearchBar = tview.NewInputField()
	app.SearchBar.SetAutocompleteFunc(func(currentText string) (entries []string) {

		if currentText == "" {
			return
		}

		var nodeNames []string

		if app.SearchBarCloneMode {
			nodeNames = app.SearchBarCloneModeAutocompleteNames
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
	app.SearchBar.SetLabel("Search Node: ")
	app.SearchBar.SetLabelColor(tcell.ColorLightBlue)
	app.SearchBar.SetFieldBackgroundColor(tcell.ColorDarkSlateBlue)
	app.SearchBar.SetFieldTextColor(tcell.ColorLightBlue)

	app.SearchBar.SetChangedFunc(func(text string) {

		if text != "" {

			if app.SearchBarCloneMode {
				res, err := app.sendRequest(newNodeCreatePacket())
				if err != nil {
					log.Println(err)
				} else if res.(*nodeCreatePacket).ViableNodes != nil {
					app.SearchBarCloneModeAutocompleteNames = res.(*nodeCreatePacket).ViableNodes
				}
			} else {

				// Hacky fix
				app.TreeView.GetRoot().ExpandAll()

				nodesToSearch := append([]sceneNode{app.currentSceneTree}, app.currentSceneTree.ChildrenRecursive()...)

				for _, node := range nodesToSearch {

					if strings.Contains(strings.ToLower(node.Name), strings.ToLower(text)) {
						treeNode := app.SceneNodesToTreeNodes[node.NodeID]
						treeNode.ExpandAll()
						app.TreeView.SetCurrentNode(treeNode)
						app.sendRequest(newNodeSelectPacket(node.NodeID))
						break
					}

				}

			}

		}

	})

	app.SearchBar.SetDoneFunc(func(key tcell.Key) {
		if app.SearchBarCloneMode && key == tcell.KeyEnter {
			np := newNodeCreatePacket()
			np.NodeToCreate = app.SearchBar.GetText()
			res, err := app.sendRequest(np)
			if err != nil {
				log.Println(err)
			} else {
				app.SelectNextNode = true
				app.currentSceneTree = res.(*nodeCreatePacket).SceneTree
				app.SelectNextNodeIndex = res.(*nodeCreatePacket).NewSelectedNode
			}
		}
		app.App.SetFocus(app.TreeView)
	})

	leftSide.AddItem(app.SearchBar, 1, 0, false)

	//

	helpText := combineStrings(
		"Welcome to TetraTerm!",
		"\n",
		"\n",
		"With TetraTerm, you can easily see and",
		"modify game data while it is running, making",
		"debugging and the flow of game development smoother.",
		"\n",
		"\n",
		"By default, TetraTerm will connect on port 7979.",
		"Just run your game after having created a TetraTerm.Server,",
		"and the terminal should pick it up automatically.",
	)

	keyText := `KEYS

Arrow Keys: Select Node
Shift+Arrows: Change Order / Parent
WASD, QE: Move Node in-game
R: Reset Selected Node
Shift+D: Duplicate Node
Shift+X: Delete Node

F: Follow Node with Camera
Shift+F: Search Nodes
Shift+C: Clone Nodes

Ctrl+Q : Quit (Ctrl+C also works)
`

	cloneText := combineStrings(
		"CLONING",
		"\n",
		"\n",
		"When you press Ctrl+C, you go into clone mode.",
		"In Clone Mode, you can specify any Node by name and clone it,",
		"reparenting it to the currently selected Node.",
		"\n",
		"\n",
		"The Node to clone can come from any Scene in the current Library.",
	)

	rightSide := tview.NewFlex()
	rightSide.SetDirection(tview.FlexRow)
	treeFlex.AddItem(rightSide, 0, 1, false)

	keysExplanation := newMultipageModal(app, "key explanation", helpText, keyText, cloneText)

	app.NodePropertyArea = tview.NewTextArea()
	app.NodePropertyArea.SetBorder(true)
	app.NodePropertyArea.SetTitle("[ Node Properties ]")
	rightSide.AddItem(app.NodePropertyArea, 0, 2, false)

	app.GamePropertyArea = tview.NewTextArea()
	app.GamePropertyArea.SetBorder(true)
	app.GamePropertyArea.SetTitle("[ Game Properties ]")
	rightSide.AddItem(app.GamePropertyArea, 0, 1, false)

	app.Root.AddAndSwitchToPage("Tree View", treeFlex, true)
	app.Root.AddPage(keysExplanation.Name, keysExplanation, true, false)

	go func() {

		for {

			time.Sleep(time.Millisecond * 250)
			app.receivingData.Store(true)

			resp, err := app.sendRequest(newSceneRefreshPacket())

			if err != nil {
				if err.Error() == "EOF" {
					// Attempt to reopen connection
					// log.Println("Request failed, attempting to reopen connection:")
					app.initClient()
				} else {
					panic(err)
				}
			} else {
				app.currentSceneTree = resp.(*sceneRefreshPacket).SceneTree

				app.currentSceneTree.ResetParenting()

				for _, node := range app.currentSceneTree.ChildrenRecursive() {

					existingNode, exists := app.SceneNodesToTreeNodes[node.NodeID]

					if !exists {
						tn := tview.NewTreeNode(node.Name)
						tn.SetSelectable(true)
						tn.SetReference(node)
						app.SceneNodesToTreeNodes[node.NodeID] = tn
						existingNode = tn
					}

					existingNode.ClearChildren()

					if node.parent != nil {
						app.SceneNodesToTreeNodes[node.parent.NodeID].AddChild(existingNode)
					}

				}

				app.TreeNodeRoot = app.SceneNodesToTreeNodes[app.currentSceneTree.NodeID]
				app.TreeNodeRoot.SetSelectable(true)
				app.TreeNodeRoot.SetColor(tcell.ColorSkyblue)
				app.TreeView.SetRoot(app.TreeNodeRoot)
				if app.TreeView.GetCurrentNode() == nil {
					app.TreeView.SetCurrentNode(app.TreeNodeRoot)
				}

				if app.SelectNextNode {
					app.TreeView.SetCurrentNode(app.SceneNodesToTreeNodes[app.SelectNextNodeIndex])
					app.SelectNextNode = false
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

			app.TreeView.SetTitle("[ ◆ Node Tree ]")

			resp, err := app.sendRequest(newNodeInfoPacket())
			if err != nil {
				log.Println(err)
			} else {
				info := resp.(*nodeInfoPacket)
				text := fmt.Sprintf("ID:%d\nVisible:%t\nType:%s\n\nPos:%v\nSca:%v\nRot:\n%v", info.ID, info.Visible, info.Type, info.Position, info.Scale, info.Rotation)
				app.App.QueueUpdateDraw(func() {
					app.NodePropertyArea.SetText(text, false)
				})
				// fmt.Println(info)
			}

			// app.receivingData.Store(true)
			// app.receivingData.Store(false)
		}
	}()

	go func() {
		for {
			time.Sleep(time.Millisecond * 200)

			resp, err := app.sendRequest(newGameInfoPacket())
			if err != nil {
				log.Println(err)
			} else {
				info := resp.(*gameInfoPacket)

				m := info.DebugInfo.AvgFrameTime.Round(time.Microsecond).Microseconds()
				ft := fmt.Sprintf("%.2fms", float32(m)/1000)

				text := fmt.Sprintf(
					"FPS:%v\nTPS:%v\nTotal Nodes: %d\nAvg. Frame-time: %s\nDrawn MeshParts: %d/%d\nDrawn Triangles: %d/%d",
					info.FPS, info.TPS,
					app.currentSceneTree.Count(),
					ft,
					info.DebugInfo.DrawnParts,
					info.DebugInfo.TotalParts,
					info.DebugInfo.DrawnTris,
					info.DebugInfo.TotalTris)

				app.App.QueueUpdate(func() {
					app.GamePropertyArea.SetText(text, false)
				})
				// fmt.Println(info)
			}

			// app.receivingData.Store(true)
			// app.receivingData.Store(false)
		}
	}()

	app.TreeView.SetChangedFunc(func(node *tview.TreeNode) {
		app.sendRequest(newNodeSelectPacket(node.GetReference().(sceneNode).NodeID))
	})

	app.TreeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// tvNode := app.TreeView.GetCurrentNode()

		// sceneNode := tvNode.GetReference().(SceneNode)

		// app.recordOriginalSettings(t3dNode.Scene().Root)

		// for _, node := range t3dNode.Scene().Root.ChildrenRecursive() {
		// 	app.recordOriginalSettings(node)
		// }

		if event.Rune() == 'f' {
			app.sendRequest(newNodeFollowCameraPacket())
			return nil
		}

		if event.Rune() == 'F' {
			app.SearchBar.SetText("")
			app.SearchBar.SetLabel("Search Node: ")
			app.SearchBarCloneMode = false
			app.App.SetFocus(app.SearchBar)
			return nil
		}

		if event.Rune() == 'C' {
			app.SearchBar.SetText("")
			app.SearchBar.SetLabel("Clone Node: ")
			app.SearchBarCloneMode = true
			app.App.SetFocus(app.SearchBar)
			return nil
		}

		if event.Rune() == 'D' {
			app.receivingData.Store(true)
			res, err := app.sendRequest(newNodeDuplicatePacket())
			if err != nil {
				log.Println(err)
			} else {
				app.SelectNextNode = true
				app.currentSceneTree = res.(*nodeDuplicatePacket).SceneTree
				app.SelectNextNodeIndex = res.(*nodeDuplicatePacket).NewSelectedNode
			}
			app.receivingData.Store(false)
			return nil
		}
		if event.Rune() == 'X' {
			app.receivingData.Store(true)
			res, err := app.sendRequest(newNodeDeletePacket())
			if err != nil {
				log.Println(err)
			} else {
				app.SelectNextNode = true
				app.currentSceneTree = res.(*nodeDeletePacket).SceneTree
				app.SelectNextNodeIndex = res.(*nodeDeletePacket).NewSelectedNode
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
				moveDir = mitMoveUp
			case tcell.KeyDown:
				moveDir = mitMoveDown
			case tcell.KeyRight:
				moveDir = mitIndent
			case tcell.KeyLeft:
				moveDir = mitDeIndent
			}

			packet := newNodeMoveInTreePacket(moveDir)

			app.receivingData.Store(true)

			res, err := app.sendRequest(packet)
			if err != nil {
				log.Println(err)
			} else {
				app.SelectNextNode = true
				app.currentSceneTree = res.(*nodeMoveInTreePacket).SceneTree
				app.SelectNextNodeIndex = res.(*nodeMoveInTreePacket).NewSelectedNode
			}

			app.receivingData.Store(false)

			return nil

		}

		// }

		moveDist := 1.0
		if event.Rune() == 'w' {
			app.sendRequest(newNodeMovePacket(0, 0, -moveDist))
			return nil
		}
		if event.Rune() == 'd' {
			app.sendRequest(newNodeMovePacket(moveDist, 0, 0))
			return nil
		}

		if event.Rune() == 's' {
			app.sendRequest(newNodeMovePacket(0, 0, moveDist))
			return nil
		}
		if event.Rune() == 'a' {
			app.sendRequest(newNodeMovePacket(-moveDist, 0, 0))
			return nil
		}

		if event.Rune() == 'q' {
			app.sendRequest(newNodeMovePacket(0, -moveDist, 0))
			return nil
		}

		if event.Rune() == 'e' {
			app.sendRequest(newNodeMovePacket(0, moveDist, 0))
			return nil
		}

		if event.Key() == tcell.KeyCtrlH {
			app.Root.ShowPage("key explanation")
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
			app.sendRequest(newNodeResetPacket())
			return nil
		}

		return event
	})

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

	if td.running.Load() {
		td.App.Sync()
	}

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

func (td Display) sendRequest(packet iPacket) (iPacket, error) {

	response, err := td.Client.Send(packet.DataType(), packet.Encode())
	if err != nil {
		return nil, err
	}

	return packet, packet.Decode(response)
	// return DecodePacket(packet.DataType(), response)

}

func (display *Display) updateTreeNodeNames() {

	for _, node := range display.currentSceneTree.ChildrenRecursive() {

		tn := display.SceneNodesToTreeNodes[node.NodeID]

		name := node.Name

		if len(node.Children) > 0 {
			if tn.IsExpanded() {
				name = "[=] " + name
			} else {
				name = "[+] " + name
			}
		} else {
			name = " ◆ " + name
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

func combineStrings(text ...string) string {
	return strings.Join(text, " ")
}
