package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// View renders the entire UI
func (m model) View() string {
	var s strings.Builder

	// title
	s.WriteString(titleStyle.Render("📤 file transfer tui"))
	s.WriteString("\n\n")

	// file browser section
	s.WriteString(sectionStyle.Render("📁 browse files"))
	s.WriteString(fmt.Sprintf(" (dir: %s)\n", m.currentDir))

	if len(m.files) == 0 {
		s.WriteString("  (no files)\n")
	} else {
		// calculate visible range
		availableHeight := m.height - 15
		visibleEnd := m.scrollPos + availableHeight
		if visibleEnd > len(m.files) {
			visibleEnd = len(m.files)
		}

		// show files in visible range
		for i := m.scrollPos; i < visibleEnd; i++ {
			f := m.files[i]
			icon := "📄"
			if f.IsDir() {
				icon = "📁"
			}

			// check if selected
			selected := ""
			fullPath := filepath.Join(m.currentDir, f.Name())
			for _, sel := range m.selectedFiles {
				if sel == fullPath {
					selected = " ✓"
					break
				}
			}

			// highlight current cursor position
			cursor := "  "
			if i == m.cursorPos {
				cursor = "> "
			}

			s.WriteString(fmt.Sprintf(
				"%s%s %s%s\n",
				cursor,
				icon,
				f.Name(),
				selected,
			))
		}

		// show scroll indicator if there are more files
		if visibleEnd < len(m.files) {
			s.WriteString(fmt.Sprintf("  ... (%d more files)\n", len(m.files)-visibleEnd))
		}
	}

	s.WriteString("\n")

	// selected files list
	if len(m.selectedFiles) > 0 {
		s.WriteString(sectionStyle.Render("✓ selected:"))
		s.WriteString("\n")
		for _, f := range m.selectedFiles {
			s.WriteString(selectedFileStyle.Render(fmt.Sprintf("  • %s", f)))
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// destination input
	s.WriteString(sectionStyle.Render("🎯 destination"))
	s.WriteString("\n")
	s.WriteString(m.targetInput.View())
	s.WriteString("\n\n")

	// transfer progress
	if m.transferring {
		s.WriteString(sectionStyle.Render("⏳ progress"))
		s.WriteString("\n")
		s.WriteString(m.progress.View())
		s.WriteString("\n\n")

		// per-file progress
		if len(m.fileProgress) > 0 {
			s.WriteString(sectionStyle.Render("📊 files:"))
			s.WriteString("\n")

			for _, fp := range m.fileProgress {
				// status icon
				icon := "⏳"
				if fp.Status == "complete" {
					icon = "✅"
				} else if fp.Status == "error" {
					icon = "❌"
				}

				// percent
				pct := ""
				if fp.TotalBytes > 0 {
					p := float64(fp.BytesTransferred) /
						float64(fp.TotalBytes) * 100
					pct = fmt.Sprintf(" %.0f%%", p)
				}

				s.WriteString(fmt.Sprintf(
					"  %s %s%s\n",
					icon,
					fp.FileName,
					pct,
				))

				// error message
				if fp.Error != "" {
					s.WriteString(errorStyle.Render(
						fmt.Sprintf("    ↳ %s\n", fp.Error),
					))
				}
			}
			s.WriteString("\n")
		}
	}

	// status line
	s.WriteString(statusStyle.Render(m.status))
	s.WriteString("\n\n")

	// help text
	if m.transferring {
		s.WriteString(helpStyle.Render(
			"transferring... ctrl+c: exit",
		))
	} else if m.focusedComponent == 0 {
		s.WriteString(helpStyle.Render(
			"↑↓: navigate • ←→: dir/select • space: select • tab: destination • q: quit",
		))
	} else {
		s.WriteString(helpStyle.Render(
			"type destination • tab: files • enter: transfer • q: quit",
		))
	}

	return s.String()
}
