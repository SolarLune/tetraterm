# TetraTerm 🖥️

![TetraTermImg](https://i.imgur.com/Tle6NgT.png)

[Devlog Video Talking About It](https://youtu.be/3Og4Ii0_QTw)

**TetraTerm** is a terminal-based debugging app for Tetra3D games and applications. It features a node graph, easy visualization of node properties, and means to create, modify, and delete nodes with minimal modifications to your game. It works by hooking into your game using localhost networking; this means your game can crash or be interrupted and you can debug as you would normally, while TetraTerm continues without issue and will re-connect when possible.

## How to install

The general approach to install TetraTerm easily would be:

1. Run `go get github.com/solarlune/tetraterm` from your game project directory. This downloads `tetraterm` so you can create a server for your game. The server runs in your game and connects to the terminal end of TetraTerm.

2. Create a `Server` instance in your game and call `Server.Update()` and `Server.Draw()` every tick from your `Game.Update()` function.

3. Run `go install github.com/solarlune/tetraterm/terminal` to install the `tetraterm` terminal display application to your Go bin directory. If that Go bin directory (which defaults to `~/go` on Linux) is in your terminal path, then you will now be able to run `tetraterm` from anywhere by just typing that command. You can also just checkout the TetraTerm repo and run or build the terminal application from the terminal directory for the same result.

4. Now just run your game as you normally do, and run `tetraterm` from a terminal. They should automatically connect to each other via port 7979 (for now, that's hardcoded into the terminal side).

See the example for more information.

## Basic usage

TetraTerm has a few options - you have the node graph at the left and the game and node properties on the right. You can use the keyboard or mouse to focus on UI elements, and collapse or expand the node tree using space or enter. Ctrl+H lists general help and key information.

## To-do

- [x] Get it to work!
- [x] Connection / Re-connection if server crashes
  - [x] Connection indicator
  - [x] Not freezing the terminal until a connection is established when one is abruptly stopped
- [ ] Quit button (modal?)
- [ ] When changing scenes, try to select a node with the same name in the new scene (i.e. if the Player
      is selected in Scene A and you go to Scene B, try to select the Player again)
- [x] Node graph
  - [ ] Hide or gray out nodes in the tree that aren't selected?
- [x] Properties panel.
  - [x] Output position, scale, rotation, ID
  - [ ] Tag display for properties
  - [ ] Keys to scale
  - [ ] Keys to rotate (this was working previously, I think it would be better to have a mode switch between moving, scaling, and rotation, though).
  - [ ] Mode switch for local vs world movement?
  - [ ] Allow modification / setting these properties numerically
- [x] Game properties panel.
  - [x] FPS, TPS, frame-time, render count
- [ ] Cloning Nodes should parent them to the currently highlighted Node 
- [x] Vertical progress bar / draggable for visualizing or even scrolling through the node tree?
  - [ ] Display scroll percentage? (not sure if I want this anymore)
- [ ] Scrolling through the node tree with mouse wheel doesn't work? (This has been determined to be an issue with tview, not TetraTerm.)
- [ ] Follow Nodes
  - [ ] Built-in free-look camera to not modify the hierarchy when following a node?
  - [ ] Following Nodes should look at the target constantly, regardless of camera movement (?)
- [ ] Keybindings
  - [ ] Expand / Collapse All
  - [ ] Expand / Collapse All Up to Current Node
  - [ ] Customizeable keybindings?
- [x] Search pane (Shift+F)
  - [ ] Search by tag, node type
  - [ ] Shift+F while a node name is entered to cycle through Nodes with those names
- [x] Clone pane (Shift+C)
  - [x] Cloning objects from other scenes in the library
- [x] Option to toggle debug drawing from terminal (1 key, by default)
- [ ] Ability to create a blank node for hierarchy-altering purposes
- [ ] Ability to track a node to always display its properties in another pane (underneath the existing Node Properties pane?)
- Flags
  - [x] Flag to change port
  - [x] Flag to change host
- [x] EOFs when terminal expects a message can cause terminal drawing distortions; the terminal needs to be cleared when this happens
- [ ] Options menu
  - [ ] Auto-hide, collapse, or darken treeview elements that are not selected?
  - [ ] Ini file for settings storage