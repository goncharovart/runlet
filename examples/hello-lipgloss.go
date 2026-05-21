// runlet:dep github.com/charmbracelet/lipgloss v1.0.0
package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)
	fmt.Println(style.Render("hello from a single file"))
}
