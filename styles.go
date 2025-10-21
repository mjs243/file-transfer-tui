package main

import "github.com/charmbracelet/lipgloss"

var (
	// color palette
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#FF6B6B")
	successColor   = lipgloss.Color("#51CF66")
	mutedColor     = lipgloss.Color("#6C757D")
	errorColor     = lipgloss.Color("#FF0000")

	// styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	selectedFileStyle = lipgloss.NewStyle().
				Foreground(successColor)

	statusStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)
)