package tetraterm

import (
	p2p "github.com/leprosus/golang-p2p"
	"github.com/solarlune/tetra3d"
)

const (
	PTSceneRefresh     = "SceneRefresh"
	PTNodeMove         = "NodeMove"
	PTNodeRotate       = "NodeRotate"
	PTNodeDuplicate    = "NodeDuplicate"
	PTNodeCreate       = "NodeCreate"
	PTNodeDelete       = "NodeDelete"
	PTNodeFollowCamera = "CameraFollow"
	PTNodeSelect       = "SelectNode"
	PTNodeReset        = "ResetNode"
	PTNodeInfo         = "NodeInfo"
	PTNodeMoveInTree   = "NodeMoveInTree"
)

type IPacket interface {
	Encode() p2p.Data
	Decode(req p2p.Data) error
	DataType() string
}

//

type SceneRefreshPacket struct {
	SceneTree SceneNode
}

func NewSceneRefreshPacket() *SceneRefreshPacket {
	return &SceneRefreshPacket{}
}

func (packet *SceneRefreshPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *SceneRefreshPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *SceneRefreshPacket) DataType() string {
	return PTSceneRefresh
}

//

type NodeMovePacket struct {
	X, Y, Z float64
}

func NewNodeMovePacket(x, y, z float64) *NodeMovePacket {
	return &NodeMovePacket{X: x, Y: y, Z: z}
}

func (packet *NodeMovePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeMovePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeMovePacket) DataType() string {
	return PTNodeMove
}

//

type NodeRotatePacket struct {
	X, Y, Z, Angle float64
}

func NewNodeRotatePacket(x, y, z, angle float64) *NodeRotatePacket {
	return &NodeRotatePacket{X: x, Y: y, Z: z, Angle: angle}
}

func (packet *NodeRotatePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeRotatePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeRotatePacket) DataType() string {
	return PTNodeRotate
}

//

type NodeSelectPacket struct {
	NodeID uint64
}

func NewNodeSelectPacket(nodeID uint64) *NodeSelectPacket {
	return &NodeSelectPacket{NodeID: nodeID}
}

func (packet *NodeSelectPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeSelectPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeSelectPacket) DataType() string {
	return PTNodeSelect
}

//

type NodeFollowCameraPacket struct{}

func NewNodeFollowCameraPacket() *NodeFollowCameraPacket {
	return &NodeFollowCameraPacket{}
}

func (packet *NodeFollowCameraPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeFollowCameraPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeFollowCameraPacket) DataType() string {
	return PTNodeFollowCamera
}

//

type NodeDuplicatePacket struct {
	NewSelectedNode uint64
	SceneTree       SceneNode
}

func NewNodeDuplicatePacket() *NodeDuplicatePacket {
	return &NodeDuplicatePacket{}
}

func (packet *NodeDuplicatePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeDuplicatePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeDuplicatePacket) DataType() string {
	return PTNodeDuplicate
}

//

type NodeDeletePacket struct {
	NewSelectedNode uint64
	SceneTree       SceneNode
}

func NewNodeDeletePacket() *NodeDeletePacket {
	return &NodeDeletePacket{}
}

func (packet *NodeDeletePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeDeletePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeDeletePacket) DataType() string {
	return PTNodeDelete
}

//

type NodeResetPacket struct{}

func NewNodeResetPacket() *NodeResetPacket {
	return &NodeResetPacket{}
}

func (packet *NodeResetPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeResetPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeResetPacket) DataType() string {
	return PTNodeReset
}

//

type NodeInfoPacket struct {
	Position tetra3d.Vector
	Scale    tetra3d.Vector
	Rotation tetra3d.Quaternion
}

func NewNodeInfoPacket() *NodeInfoPacket {
	return &NodeInfoPacket{}
}

func (packet *NodeInfoPacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeInfoPacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeInfoPacket) DataType() string {
	return PTNodeInfo
}

//

const (
	MITIndent = iota
	MITDeIndent
	MITMoveUp
	MITMoveDown
)

type NodeMoveInTreePacket struct {
	MoveDir         int
	NewSelectedNode uint64
	SceneTree       SceneNode
}

func NewNodeMoveInTreePacket(moveDir int) *NodeMoveInTreePacket {
	return &NodeMoveInTreePacket{MoveDir: moveDir}
}

func (packet *NodeMoveInTreePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeMoveInTreePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeMoveInTreePacket) DataType() string {
	return PTNodeMoveInTree
}

//

type NodeCreatePacket struct {
	NodeToCreate string

	ViableNodes []string

	NewSelectedNode uint64
	SceneTree       SceneNode
}

func NewNodeCreatePacket() *NodeCreatePacket {
	return &NodeCreatePacket{}
}

func (packet *NodeCreatePacket) Encode() p2p.Data {
	data := p2p.Data{}
	err := data.SetGob(packet)
	if err != nil {
		panic(err)
	}
	return data
}

func (packet *NodeCreatePacket) Decode(req p2p.Data) error {
	return req.GetGob(&packet)
}

func (packet *NodeCreatePacket) DataType() string {
	return PTNodeCreate
}
