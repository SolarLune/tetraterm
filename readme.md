# TetraTerm 🖥️

**TetraTerm** is a terminal-based debugging app for Tetra3D games and applications. It features a node graph, easy visualization of node properties, and means to create, modify, and delete nodes with minimal modifications to your game. It works by hooking into your game using networking; this means your game can crash or be interrupted and you can debug as you would like, while TetraTerm continues without issue and will re-connect when possible.

## How to use

The general approach to make this easiest to use would be to do as follows:

Firstly, run `go get github.com/solarlune/tetraterm`. This downloads `tetraterm` so you can create a server for your game project.

2. Run `go install github.com/solarlune/tetraterm/terminal`

3. Create a `Server` instance in your game and update it every frame.
4. Run the game.
5. Run `tetraterm` in a terminal. It should automatically read your game while it's running and connect to your game process via port 8000 to your game.

## To-do

- [x] Get it to work
- [x] Connection / Re-connection if server crashes
- [x] Node graph
- [x] Add Properties panel.
  - [x] Output position, scale, rotation, ID
  - [ ] Allow modification of these properties?
- [x] Search pane (Shift+F)
- [x] Clone pane (Shift+C)
- [ ] 