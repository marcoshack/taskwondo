package service

import (
	"testing"

	"github.com/marcoshack/taskwondo/internal/model"
)

func TestExtractTeamText(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		desc := "Handles backend services"
		team := &model.Team{Name: "Backend", Description: &desc}
		got := extractTeamText(team)
		want := "Team: Backend\n\nHandles backend services"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("without description", func(t *testing.T) {
		team := &model.Team{Name: "Frontend"}
		got := extractTeamText(team)
		want := "Team: Frontend"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty description", func(t *testing.T) {
		desc := ""
		team := &model.Team{Name: "QA", Description: &desc}
		got := extractTeamText(team)
		want := "Team: QA"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
