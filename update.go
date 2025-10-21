package main

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Init runs when the app starts
func (m model) Init() tea.Cmd {
	return tickCmd()
}

// tickCmd returns a command that ticks periodically
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update processes all messages
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// always handle quit
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case TransferProgressMessage:
		m = m.handleProgress(msg)
		return m, nil

	case TransferErrorMessage:
		m.status = fmt.Sprintf("❌ error: %s", msg.Error)
		m.transferring = false
		return m, nil

	case TransferCompleteMessage:
		m.transferring = false
		m.status = "✅ transfer complete!"
		m.selectedFiles = []string{}
		return m, nil

	case transferStartedMsg:
		m.progressChan = msg.progressChan
		m.status = "🚀 transferring..."
		return m, tickCmd()

	case TickMsg:
		// check for progress updates from transfer
		if m.progressChan != nil {
			select {
			case msg, ok := <-m.progressChan:
				if !ok {
					// channel closed = transfer complete
					m.progressChan = nil
					m.transferring = false
					m.status = "✅ transfer complete!"
					m.selectedFiles = []string{}
					return m, nil
				}
				m = m.handleProgress(msg)
			default:
				// no message available yet
			}
		}
		return m, tickCmd()
	}

	// if transferring, don't process other input
	if m.transferring {
		return m, tickCmd()
	}

	// normal operation mode
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "tab":
			// switch focus between file browser and input
			m.focusedComponent = (m.focusedComponent + 1) % 2

		case "up":
			if m.focusedComponent == 0 && m.cursorPos > 0 {
				m.cursorPos--
				m.ensureCursorVisible()
			}

		case "down":
			if m.focusedComponent == 0 && m.cursorPos < len(m.files)-1 {
				m.cursorPos++
				m.ensureCursorVisible()
			}

		case "left":
			// go up one directory
			if m.focusedComponent == 0 {
				parent := filepath.Dir(m.currentDir)
				if parent != m.currentDir {
					m.currentDir = parent
					m.loadFiles()
					m.cursorPos = 0
				}
			}

		case "right":
			// enter directory or select file
			if m.focusedComponent == 0 && len(m.files) > 0 {
				currentFile := m.files[m.cursorPos]
				fullPath := filepath.Join(m.currentDir, currentFile.Name())

				if currentFile.IsDir() {
					m.currentDir = fullPath
					m.loadFiles()
					m.cursorPos = 0
				} else {
					m.toggleSelection(fullPath)
					m.status = fmt.Sprintf(
						"📦 %d file(s) selected",
						len(m.selectedFiles),
					)
				}
			}

		case " ":
			// space selects file
			if m.focusedComponent == 0 && len(m.files) > 0 {
				currentFile := m.files[m.cursorPos]
				if !currentFile.IsDir() {
					fullPath := filepath.Join(m.currentDir, currentFile.Name())
					m.toggleSelection(fullPath)
					m.status = fmt.Sprintf(
						"📦 %d file(s) selected",
						len(m.selectedFiles),
					)
				}
			}

		case "enter":
			// start transfer (only from input field)
			if m.focusedComponent == 1 {
				if len(m.selectedFiles) == 0 {
					m.status = "⚠️ select files first"
					return m, nil
				}

				if m.targetInput.Value() == "" {
					m.status = "⚠️ enter destination"
					return m, nil
				}

				// begin transfer
				m.transferring = true
				m.status = "🔗 connecting..."
				m.fileProgress = make(map[string]*FileProgress)

				// init progress tracking for each file
				for _, f := range m.selectedFiles {
					m.fileProgress[f] = &FileProgress{
						FileName: filepath.Base(f),
						Status:   "pending",
					}
				}

				// start the transfer command
				cmd = m.startTransfer()
				return m, cmd
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// forward message to input if focused
	if m.focusedComponent == 1 {
		m.targetInput, cmd = m.targetInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	cmds = append(cmds, tickCmd())
	return m, tea.Batch(cmds...)
}

// ensureCursorVisible keeps the cursor within the visible viewport
func (m *model) ensureCursorVisible() {
	if m.focusedComponent != 0 {
		return
	}

	// available height for file list (rough estimate)
	// subtract space for title, sections, input, help
	availableHeight := m.height - 15

	// if cursor is above scroll position, scroll up
	if m.cursorPos < m.scrollPos {
		m.scrollPos = m.cursorPos
	}

	// if cursor is below visible area, scroll down
	if m.cursorPos >= m.scrollPos+availableHeight {
		m.scrollPos = m.cursorPos - availableHeight + 1
	}
}

// toggleSelection adds or removes a file from selection
func (m *model) toggleSelection(path string) {
	for i, f := range m.selectedFiles {
		if f == path {
			// already selected, remove it
			m.selectedFiles = append(
				m.selectedFiles[:i],
				m.selectedFiles[i+1:]...,
			)
			return
		}
	}
	// not selected, add it
	m.selectedFiles = append(m.selectedFiles, path)
}

// handleProgress updates file progress and overall bar
func (m model) handleProgress(msg TransferProgressMessage) model {
	if _, ok := m.fileProgress[msg.FileName]; !ok {
		m.fileProgress[msg.FileName] = &FileProgress{
			FileName: msg.FileName,
		}
	}

	fp := m.fileProgress[msg.FileName]
	fp.Status = msg.Status
	fp.BytesTransferred = msg.BytesTransferred
	fp.TotalBytes = msg.TotalBytes
	fp.Error = msg.Error

	// calculate overall progress
	var totalBytes, completedBytes int64

	for _, p := range m.fileProgress {
		totalBytes += p.TotalBytes
		if p.Status == "complete" {
			completedBytes += p.TotalBytes
		} else if p.Status == "transferring" {
			completedBytes += p.BytesTransferred
		}
	}

	if totalBytes > 0 {
		pct := float64(completedBytes) / float64(totalBytes)
		m.progress.SetPercent(pct)
	}

	return m
}

// startTransfer initiates a file transfer
func (m model) startTransfer() tea.Cmd {
	selectedFiles := m.selectedFiles
	destination := m.targetInput.Value()

	return func() tea.Msg {
		// parse destination
		user, host, remotePath, err := parseDestination(destination)
		if err != nil {
			return TransferErrorMessage{Error: err.Error()}
		}

		// create ssh client
		client := NewSSHClient(SSHConfig{
			Host: host,
			User: user,
			Port: "22",
		})

		// connect
		if err := client.Connect(); err != nil {
			return TransferErrorMessage{Error: err.Error()}
		}
		defer client.Close()

		// build file map (local -> remote name)
		fileMap := make(map[string]string)
		for _, f := range selectedFiles {
			fileMap[f] = filepath.Base(f)
		}

		// create buffered progress channel
		progressChan := make(chan TransferProgressMessage, 100)

		// start transfer in background goroutine
		go client.TransferFiles(fileMap, remotePath, progressChan)

		// return message carrying the channel back to model
		return transferStartedMsg{progressChan: progressChan}
	}
}
