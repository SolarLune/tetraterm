package main

import (
	"bytes"
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"

	_ "embed"

	"github.com/solarlune/tetraterm"
)

//go:embed example.gltf
var gltfData []byte

type Game struct {
	Library *tetra3d.Library
	Scene   *tetra3d.Scene
	Camera  *tetra3d.Camera

	DebugServer *tetraterm.Server
}

func NewGame() *Game {
	g := &Game{}
	g.Init()
	return g
}

func (g *Game) Init() {

	// For this example, we'll load a GLTF file as you might normally do...

	reader := bytes.NewReader(gltfData)

	library, err := tetra3d.LoadGLTFData(reader, nil)
	if err != nil {
		panic(err)
	}

	g.Library = library
	g.Scene = g.Library.ExportedScene
	g.Camera = g.Scene.Root.Get("Camera").(*tetra3d.Camera)

	// ...But at the end, we'll create a tetraterm.Server. This is what allows
	// the connection between our game and TetraTerm. We pass nil for the
	// and specify the starting scene.
	g.DebugServer = tetraterm.NewServer(nil)

}

func (g *Game) Update() error {

	cube := g.Scene.Root.Get("Cube")
	cube.Rotate(0, 1, 0, 0.01)

	var err error

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Don't forget to update the server!
	g.DebugServer.Update(g.Scene)

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	screen.Fill(colors.DarkestGray().ToRGBA64())

	g.Camera.Clear()
	g.Camera.RenderScene(g.Scene)
	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// We also call DebugServer.Draw() to handle debug mode rendering.
	g.DebugServer.Draw(screen, g.Camera)

}

func (g *Game) Layout(w, h int) (int, int) {
	return g.Camera.Size()
}

func main() {

	g := NewGame()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("TetraTerm Example")

	err := ebiten.RunGame(g)

	if err != nil && err.Error() != "quit" {
		panic(err)
	}

}
