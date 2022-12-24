package main

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/solarlune/tetra3d"
	"github.com/solarlune/tetra3d/colors"

	"github.com/solarlune/tetraterm"
)

type Game struct {
	Scene  *tetra3d.Scene
	Camera *tetra3d.Camera

	DebugServer *tetraterm.Server
}

func NewGame() *Game {
	g := &Game{}
	g.Init()
	return g
}

func (g *Game) Init() {

	// For this example, let's just say we create a scene as you might normally do...

	lib := tetra3d.NewLibrary()
	g.Scene = lib.AddScene("Test")

	g.Camera = tetra3d.NewCamera(640, 360)
	g.Camera.Move(0, 2, 10)
	g.Scene.Root.AddChildren(g.Camera)

	cubeMesh := tetra3d.NewCubeMesh()
	cube := tetra3d.NewModel(cubeMesh, "Cube")
	g.Scene.Root.AddChildren(cube)

	prismMesh := tetra3d.NewPrismMesh()
	prism := tetra3d.NewModel(prismMesh, "Prism")
	prism.Move(3, 0, -4)
	prism.Color.Set(0, 0.4, 1, 1)
	g.Scene.Root.AddChildren(prism)

	otherMesh := tetra3d.NewIcosphereMesh(1)
	sphere := tetra3d.NewModel(otherMesh, "Icosphere")
	sphere.Color.Set(1, 0, 0, 1)
	sphere.Move(0, 5, 0)
	cube.AddChildren(sphere)

	light := tetra3d.NewPointLight("point light", 1, 1, 1, 2)
	light.Move(2, 4, 2)
	g.Scene.Root.AddChildren(light)

	g.Scene.World.AmbientLight.Energy = 0.2

	assets := lib.AddScene("Assets")
	asset := tetra3d.NewModel(prismMesh, "It's A Snake, IDK")
	asset.Color.Set(1, 0, 0, 1)

	piece := asset.Clone()
	piece.SetName("snake piece")

	for i := 0; i < 5; i++ {
		p := piece.Clone()
		p.Move(0, float64(i), 0)
		asset.AddChildren(p)
	}
	assets.Root.AddChildren(asset)

	// But at the end, we create a debug server. We use the default connection settings,
	// and specify the starting scene.
	g.DebugServer = tetraterm.NewServer(nil, g.Scene)

}

func (g *Game) Update() error {

	cube := g.Scene.Root.Get("Cube")
	cube.Rotate(0, 0.7, 0.3, 0.01)

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.Scene.Root.Get("Prism").Unparent()
	}

	var err error

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		err = errors.New("quit")
	}

	// Don't forget to update the server.
	g.DebugServer.Update(g.Scene)

	return err
}

func (g *Game) Draw(screen *ebiten.Image) {

	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.Camera.Get("Blaot").Transform()
	}

	screen.Fill(colors.DarkestGray().ToRGBA64())

	g.Camera.Clear()

	g.Camera.RenderScene(g.Scene)

	screen.DrawImage(g.Camera.ColorTexture(), nil)

	// We also call DebugServer.Draw() to handle debug mode rendering.
	g.DebugServer.Draw(screen, g.Camera)

}

func (g *Game) Layout(w, h int) (int, int) {
	return 640, 360
}

func main() {

	g := NewGame()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Tetra3d-Terminal")

	err := ebiten.RunGame(g)

	if err != nil && err.Error() != "quit" {
		panic(err)
	}

}

// package main

// import (
// 	"errors"

// 	"github.com/hajimehoshi/ebiten/v2"
// 	"github.com/hajimehoshi/ebiten/v2/inpututil"
// 	"github.com/solarlune/tetra3d"
// 	"github.com/solarlune/tetra3d/colors"
// )

// type Game struct {
// 	Scene  *tetra3d.Scene
// 	Camera *tetra3d.Camera

// 	DebugTerm *TerminalApp
// }

// func NewGame() *Game {
// 	g := &Game{}
// 	g.Init()
// 	return g
// }

// func (g *Game) Init() {

// 	g.Scene = tetra3d.NewScene("Test")
// 	g.Camera = tetra3d.NewCamera(640, 360)
// 	g.Camera.Move(0, 2, 10)
// 	g.Scene.Root.AddChildren(g.Camera)

// 	cubeMesh := tetra3d.NewCubeMesh()
// 	cube := tetra3d.NewModel(cubeMesh, "Cube")
// 	g.Scene.Root.AddChildren(cube)

// 	otherMesh := tetra3d.NewIcosphereMesh(1)
// 	sphere := tetra3d.NewModel(otherMesh, "Icosphere")
// 	sphere.Color.Set(1, 0, 0, 1)
// 	sphere.Move(0, 5, 0)
// 	cube.AddChildren(sphere)

// 	light := tetra3d.NewPointLight("point light", 1, 1, 1, 2)
// 	light.Move(2, 4, 2)
// 	g.Scene.Root.AddChildren(light)

// 	g.Scene.World.AmbientLight.Energy = 0.2

// 	g.DebugTerm = NewTerminalApp(g.Camera)

// }

// func (g *Game) Update() error {

// 	defer g.DebugTerm.Recover()

// 	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
// 		g.Scene.Root.Get("blahdeblah").Transform()
// 	}

// 	cube := g.Scene.Root.Get("Cube")
// 	cube.Rotate(0, 0.7, 0.3, 0.01)

// 	var err error

// 	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
// 		err = errors.New("quit")
// 	}

// 	return err
// }

// func (g *Game) Draw(screen *ebiten.Image) {

// 	defer g.DebugTerm.Recover()

// 	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
// 		g.Camera.Get("Blaot").Transform()
// 	}

// 	screen.Fill(colors.DarkestGray().ToRGBA64())

// 	g.Camera.Clear()

// 	g.Camera.RenderScene(g.Scene)

// 	screen.DrawImage(g.Camera.ColorTexture(), nil)

// 	g.DebugTerm.Draw(screen, g.Scene)

// }

// func (g *Game) Layout(w, h int) (int, int) {
// 	return 640, 360
// }

// func main() {

// 	g := NewGame()

// 	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
// 	ebiten.SetWindowTitle("Tetra3d-Terminal")

// 	err := ebiten.RunGame(g)

// 	g.DebugTerm.Stop()

// 	if err != nil && err.Error() != "quit" {
// 		panic(err)
// 	}

// }
