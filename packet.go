package tetraterm

import (
	"strconv"

	p2p "github.com/leprosus/golang-p2p"
	"github.com/solarlune/tetra3d"
)

const (
	ptSceneRefresh             = "SceneRefresh"
	ptNodeMove                 = "NodeMove"
	ptNodeRotate               = "NodeRotate"
	ptNodeDuplicate            = "NodeDuplicate"
	ptNodeCreate               = "NodeCreate"
	ptGameInfo                 = "GameInfo"
	ptNodeDelete               = "NodeDelete"
	ptNodeFollowCamera         = "CameraFollow"
	ptNodeSelect               = "SelectNode"
	ptNodeReset                = "ResetNode"
	ptNodeInfo                 = "NodeInfo"
	ptNodeMoveInTree           = "NodeMoveInTree"
	ptToggleDebugDrawHierarchy = "ToggleDebugDrawHierarchy"
	ptToggleDebugDrawWireframe = "ToggleDebugDrawWireframe"
	ptToggleDebugDrawBounds    = "ToggleDebugDrawBounds"
)

type iPacket interface {
	Encode() p2p.Data
	Decode(req p2p.Data) error
	DataType() string
}

//

type sceneRefreshPacket struct {
	SceneTree sceneNode
}

func newSceneRefreshPacket() *sceneRefreshPacket {
	return &sceneRefreshPacket{}
}

func (packet *sceneRefreshPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *sceneRefreshPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *sceneRefreshPacket) DataType() string {
	return ptSceneRefresh
}

//

type nodeMovePacket struct {
	X, Y, Z float64
}

func newNodeMovePacket(x, y, z float64) *nodeMovePacket {
	return &nodeMovePacket{X: x, Y: y, Z: z}
}

func (packet *nodeMovePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeMovePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeMovePacket) DataType() string {
	return ptNodeMove
}

//

type nodeRotatePacket struct {
	X, Y, Z, Angle float64
}

func newNodeRotatePacket(x, y, z, angle float64) *nodeRotatePacket {
	return &nodeRotatePacket{X: x, Y: y, Z: z, Angle: angle}
}

func (packet *nodeRotatePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeRotatePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeRotatePacket) DataType() string {
	return ptNodeRotate
}

//

type nodeSelectPacket struct {
	NodeID uint64
}

func newNodeSelectPacket(nodeID uint64) *nodeSelectPacket {
	return &nodeSelectPacket{NodeID: nodeID}
}

func (packet *nodeSelectPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeSelectPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeSelectPacket) DataType() string {
	return ptNodeSelect
}

//

type nodeFollowCameraPacket struct{}

func newNodeFollowCameraPacket() *nodeFollowCameraPacket {
	return &nodeFollowCameraPacket{}
}

func (packet *nodeFollowCameraPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeFollowCameraPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeFollowCameraPacket) DataType() string {
	return ptNodeFollowCamera
}

//

type nodeDuplicatePacket struct {
	NewSelectedNode uint64
	SceneTree       sceneNode
}

func newNodeDuplicatePacket() *nodeDuplicatePacket {
	return &nodeDuplicatePacket{}
}

func (packet *nodeDuplicatePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeDuplicatePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeDuplicatePacket) DataType() string {
	return ptNodeDuplicate
}

//

type nodeDeletePacket struct {
	NewSelectedNode uint64
	SceneTree       sceneNode
}

func newNodeDeletePacket() *nodeDeletePacket {
	return &nodeDeletePacket{}
}

func (packet *nodeDeletePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeDeletePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeDeletePacket) DataType() string {
	return ptNodeDelete
}

//

type nodeResetPacket struct{}

func newNodeResetPacket() *nodeResetPacket {
	return &nodeResetPacket{}
}

func (packet *nodeResetPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeResetPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeResetPacket) DataType() string {
	return ptNodeReset
}

//

type matrix3 [3][3]float64

func matrix4ToMatrix3(mat tetra3d.Matrix4) matrix3 {
	return matrix3{
		{mat[0][0], mat[0][1], mat[0][2]},
		{mat[1][0], mat[1][1], mat[1][2]},
		{mat[2][0], mat[2][1], mat[2][2]},
	}
}

func (mat matrix3) String() string {
	str := ""

	for i := 0; i < 3; i++ {

		if i == 0 {
			str += "X: {"
		} else if i == 1 {
			str += "Y: {"
		} else {
			str += "Z: {"
		}

		for j := 0; j < 3; j++ {
			str += strconv.FormatFloat(mat[i][j], 'f', 2, 64)
			if j < 2 {
				str += ", "
			}
		}

		str += "}"

		if i < 2 {
			str += "\n"
		}

	}

	str += ""

	return str
}

type nodeInfoPacket struct {
	ID       uint64
	Position tetra3d.Vector
	Scale    tetra3d.Vector
	Rotation matrix3
	Visible  bool
	Type     tetra3d.NodeType
}

func newNodeInfoPacket() *nodeInfoPacket {
	return &nodeInfoPacket{}
}

func (packet *nodeInfoPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeInfoPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeInfoPacket) DataType() string {
	return ptNodeInfo
}

//

const (
	mitIndent = iota
	mitDeIndent
	mitMoveUp
	mitMoveDown
)

type nodeMoveInTreePacket struct {
	MoveDir         int
	NewSelectedNode uint64
	SceneTree       sceneNode
}

func newNodeMoveInTreePacket(moveDir int) *nodeMoveInTreePacket {
	return &nodeMoveInTreePacket{MoveDir: moveDir}
}

func (packet *nodeMoveInTreePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeMoveInTreePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeMoveInTreePacket) DataType() string {
	return ptNodeMoveInTree
}

//

type nodeCreatePacket struct {
	NodeToCreate string

	ViableNodes []string

	NewSelectedNode uint64
	SceneTree       sceneNode
}

func newNodeCreatePacket() *nodeCreatePacket {
	return &nodeCreatePacket{}
}

func (packet *nodeCreatePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *nodeCreatePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *nodeCreatePacket) DataType() string {
	return ptNodeCreate
}

/////

type gameInfoPacket struct {
	FPS, TPS  float32
	DebugInfo tetra3d.DebugInfo
}

func newGameInfoPacket() *gameInfoPacket {
	return &gameInfoPacket{}
}

func (packet *gameInfoPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *gameInfoPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *gameInfoPacket) DataType() string {
	return ptGameInfo
}

/////

type toggleDebugDrawHierarchy struct {
	DebugDrawOn bool
}

func newToggleDebugDrawHierarchy() *toggleDebugDrawHierarchy {
	return &toggleDebugDrawHierarchy{}
}

func (packet *toggleDebugDrawHierarchy) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *toggleDebugDrawHierarchy) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *toggleDebugDrawHierarchy) DataType() string {
	return ptToggleDebugDrawHierarchy
}

/////

type toggleDebugDrawWireframe struct {
	DebugDrawOn bool
}

func newToggleDebugDrawWireframe() *toggleDebugDrawWireframe {
	return &toggleDebugDrawWireframe{}
}

func (packet *toggleDebugDrawWireframe) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *toggleDebugDrawWireframe) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *toggleDebugDrawWireframe) DataType() string {
	return ptToggleDebugDrawWireframe
}

/////

type toggleDebugDrawBounds struct {
	DebugDrawOn bool
}

func newToggleDebugDrawBounds() *toggleDebugDrawBounds {
	return &toggleDebugDrawBounds{}
}

func (packet *toggleDebugDrawBounds) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *toggleDebugDrawBounds) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *toggleDebugDrawBounds) DataType() string {
	return ptToggleDebugDrawBounds
}
