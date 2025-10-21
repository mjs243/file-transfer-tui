package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
)

// TransferProgressMessage sent during file transfers
type TransferProgressMessage struct {
	FileName         string
	Status           string // pending, transferring, complete, error
	BytesTransferred int64
	TotalBytes       int64
	Error            string
}

// TransferErrorMessage for transfer errors
type TransferErrorMessage struct {
	Error string
}

// TransferCompleteMessage when all transfers finish
type TransferCompleteMessage struct {
	Success  bool
	ErrorMsg string
}

// transferStartedMsg carries the progress channel back to model
type transferStartedMsg struct {
	progressChan chan TransferProgressMessage
}

// FileProgress tracks a single file's transfer
type FileProgress struct {
	FileName         string
	BytesTransferred int64
	TotalBytes       int64
	Status           string
	Error            string
}

// TickMsg is sent periodically
type TickMsg time.Time

// model is the root application state
type model struct {
	// UI components
	targetInput textinput.Model
	progress    progress.Model

	// application state
	selectedFiles    []string
	transferring     bool
	status           string
	err              error
	fileProgress     map[string]*FileProgress
	focusedComponent int
	width            int
	height           int

	// file browser state
	currentDir string
	files      []os.FileInfo
	cursorPos  int
	scrollPos  int // tracks which file is at the top of the viewport

	// transfer tracking
	progressChan chan TransferProgressMessage
}

// initialModel creates the starting state
func initialModel() model {
	// target input
	ti := textinput.New()
	ti.Placeholder = "user@host:/path/to/destination"
	ti.CharLimit = 256
	ti.Width = 50
	ti.Focus()

	// progress bar
	prog := progress.New(progress.WithDefaultGradient())

	// start directory - use Documents if available, otherwise home
	startDir := homeDir()
	docsDir := filepath.Join(startDir, "Documents")
	if _, err := os.Stat(docsDir); err == nil {
		startDir = docsDir
	}

	return model{
		targetInput:      ti,
		progress:         prog,
		selectedFiles:    []string{},
		status:           "📦 ready to transfer",
		fileProgress:     make(map[string]*FileProgress),
		focusedComponent: 0, // start with file browser
		currentDir:       startDir,
		files:            []os.FileInfo{},
		cursorPos:        0,
		scrollPos:        0,
	}
}

// homeDir gets the user's home directory
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// loadFiles loads files from currentDir
func (m *model) loadFiles() error {
	entries, err := os.ReadDir(m.currentDir)
	if err != nil {
		// if we can't read this dir, try the parent
		parent := filepath.Dir(m.currentDir)
		if parent != m.currentDir {
			m.currentDir = parent
			return m.loadFiles()
		}
		return err
	}

	m.files = []os.FileInfo{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil {
			m.files = append(m.files, info)
		}
	}

	if m.cursorPos >= len(m.files) {
		m.cursorPos = 0
	}

	m.scrollPos = 0

	return nil
}
