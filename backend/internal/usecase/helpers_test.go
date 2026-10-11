package usecase

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

func newAdventurer(name string) *adventurer.Adventurer {
	return adventurer.Reconstruct(uuid.Must(uuid.NewV7()), name, testNow)
}

func openQuest(t *testing.T, title string, deadline *time.Time, createdAt time.Time) *quest.Quest {
	t.Helper()
	q, err := quest.NewQuest(uuid.Must(uuid.NewV7()), title, "", 100, deadline, createdAt)
	if err != nil {
		t.Fatalf("NewQuest: %v", err)
	}
	return q
}

// acceptedQuest は assignee が受注済みのクエストを返す。
func acceptedQuest(t *testing.T, assignee uuid.UUID) *quest.Quest {
	t.Helper()
	acceptedAt := testNow.Add(-time.Hour)
	return quest.Reconstruct(uuid.Must(uuid.NewV7()), "受注済", "", 100, nil, quest.StatusAccepted,
		&assignee, &acceptedAt, nil, testNow.Add(-2*time.Hour), 1)
}
