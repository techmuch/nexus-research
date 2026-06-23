package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestExperiment(t *testing.T) {
	m := NewModel(":memory:")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return bytes.Contains(b, []byte("User Management")) }, teatest.WithCheckInterval(time.Millisecond*50), teatest.WithDuration(time.Second*2))
	
	// enter user list
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return bytes.Contains(b, []byte("User Account Directory")) }, teatest.WithCheckInterval(time.Millisecond*50), teatest.WithDuration(time.Second*2))

	// press /
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return bytes.Contains(b, []byte("🔍 █")) }, teatest.WithCheckInterval(time.Millisecond*50), teatest.WithDuration(time.Second*2))
}
