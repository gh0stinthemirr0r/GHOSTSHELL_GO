package ui

import (
	"fmt"
	"time"

	"yourproject/sshmgmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// SSHManagerUI handles the graphical interface for managing SSH connections.
type SSHManagerUI struct {
	sshManager *sshmgmt.SSHManager

	x, y, width, height int32
	font                rl.Font
	connections         []sshmgmt.SSHConnection
	selectedConnection  int

	// UI Elements
	saveButton   Button
	deleteButton Button
	tempButton   Button
	listBox      ListBox
}

// NewSSHManagerUI initializes a new SSHManagerUI instance.
func NewSSHManagerUI(sshManager *sshmgmt.SSHManager, font rl.Font, x, y, width, height int32) *SSHManagerUI {
	return &SSHManagerUI{
		sshManager:   sshManager,
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		font:         font,
		connections:  sshManager.ListConnections(),
		listBox:      NewListBox(x+20, y+60, width-40, height-140, font),
		saveButton:   NewButton(x+20, y+height-60, 120, 40, "Save"),
		deleteButton: NewButton(x+160, y+height-60, 120, 40, "Delete"),
		tempButton:   NewButton(x+300, y+height-60, 120, 40, "Temp Connect"),
	}
}

// Update handles user interactions with the UI.
func (ui *SSHManagerUI) Update() {
	// Update list of connections
	ui.connections = ui.sshManager.ListConnections()
	ui.listBox.Update(ui.connections)

	// Handle button presses
	if ui.saveButton.Update() {
		ui.handleSaveConnection()
	}

	if ui.deleteButton.Update() {
		ui.handleDeleteConnection()
	}

	if ui.tempButton.Update() {
		ui.handleTempConnection()
	}
}

// Draw renders the SSH Manager UI.
func (ui *SSHManagerUI) Draw() {
	rl.ClearBackground(rl.DarkGray)

	// Draw Title
	rl.DrawTextEx(ui.font, "SSH Manager", rl.Vector2{X: float32(ui.x + 20), Y: float32(ui.y + 10)}, 24, 2, rl.White)

	// Draw List Box
	ui.listBox.Draw()

	// Draw Buttons
	ui.saveButton.Draw()
	ui.deleteButton.Draw()
	ui.tempButton.Draw()
}

// handleSaveConnection handles saving a new SSH connection.
func (ui *SSHManagerUI) handleSaveConnection() {
	name := fmt.Sprintf("Connection_%d", time.Now().Unix())
	address := "example.com:22"
	username := "user"
	privateKey := "PRIVATE_KEY_PLACEHOLDER"

	err := ui.sshManager.SaveConnection(name, address, username, privateKey, false)
	if err != nil {
		fmt.Println("Failed to save connection:", err)
		return
	}
	fmt.Println("Connection saved:", name)
}

// handleDeleteConnection handles deleting the selected SSH connection.
func (ui *SSHManagerUI) handleDeleteConnection() {
	if ui.listBox.SelectedIndex >= 0 && ui.listBox.SelectedIndex < len(ui.connections) {
		selected := ui.connections[ui.listBox.SelectedIndex]
		if err := ui.sshManager.DeleteConnection(selected.Name); err != nil {
			fmt.Println("Failed to delete connection:", err)
			return
		}
		fmt.Println("Connection deleted:", selected.Name)
	}
}

// handleTempConnection creates a temporary connection.
func (ui *SSHManagerUI) handleTempConnection() {
	tempAddress := "temp.example.com:22"
	tempUsername := "tempUser"
	tempPrivateKey := "TEMP_PRIVATE_KEY"

	conn, err := ui.sshManager.CreateTempConnection(tempAddress, tempUsername, tempPrivateKey)
	if err != nil {
		fmt.Println("Failed to create temporary connection:", err)
		return
	}
	fmt.Printf("Temporary connection created: %s -> %s\n", conn.Name, conn.Address)
}

// ListBox represents a scrollable list of items.
type ListBox struct {
	x, y, width, height int32
	font                rl.Font
	items               []sshmgmt.SSHConnection
	SelectedIndex       int
}

// NewListBox initializes a new ListBox.
func NewListBox(x, y, width, height int32, font rl.Font) ListBox {
	return ListBox{
		x:      x,
		y:      y,
		width:  width,
		height: height,
		font:   font,
	}
}

// Update updates the list box items and handles scrolling/selection.
func (lb *ListBox) Update(items []sshmgmt.SSHConnection) {
	lb.items = items
	if rl.IsKeyPressed(rl.KeyUp) {
		lb.SelectedIndex--
		if lb.SelectedIndex < 0 {
			lb.SelectedIndex = 0
		}
	} else if rl.IsKeyPressed(rl.KeyDown) {
		lb.SelectedIndex++
		if lb.SelectedIndex >= len(lb.items) {
			lb.SelectedIndex = len(lb.items) - 1
		}
	}
}

// Draw renders the list box.
func (lb *ListBox) Draw() {
	rl.DrawRectangle(lb.x, lb.y, lb.width, lb.height, rl.Black)
	rl.DrawRectangleLines(lb.x, lb.y, lb.width, lb.height, rl.White)

	visibleItems := int(lb.height) / 24 // Approx. height of a row
	startIdx := 0
	if lb.SelectedIndex >= visibleItems {
		startIdx = lb.SelectedIndex - visibleItems + 1
	}

	for i, item := range lb.items[startIdx:] {
		if i >= visibleItems {
			break
		}
		rowY := lb.y + int32(i*24)
		color := rl.Gray
		if startIdx+i == lb.SelectedIndex {
			color = rl.White
		}
		rl.DrawTextEx(lb.font, fmt.Sprintf("%s (%s)", item.Name, item.Address), rl.Vector2{X: float32(lb.x + 5), Y: float32(rowY)}, 20, 2, color)
	}
}
