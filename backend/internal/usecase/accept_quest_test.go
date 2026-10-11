package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

func TestAcceptQuest(t *testing.T) {
	a := newAdventurer("アリア")
	q := openQuest(t, "薬草採取", nil, testNow.Add(-time.Hour))
	repo := newFakeQuestRepo(q)
	uc := NewAcceptQuest(repo, newFakeAdventurerRepo(a), FixedClock{T: testNow})

	v, err := uc.Execute(context.Background(), q.ID(), a.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Quest.Status() != quest.StatusAccepted || v.Assignee == nil || v.Assignee.ID() != a.ID() {
		t.Errorf("unexpected view: status=%s assignee=%v", v.Quest.Status(), v.Assignee)
	}
	if !v.Quest.AcceptedAt().Equal(testNow) {
		t.Errorf("acceptedAt = %v, want %v", v.Quest.AcceptedAt(), testNow)
	}
	if v.Quest.Version() != 2 {
		t.Errorf("version = %d, want 2 (Update must be called)", v.Quest.Version())
	}
}

func TestAcceptQuest_Errors(t *testing.T) {
	a := newAdventurer("アリア")
	past := testNow.Add(-time.Minute)
	tests := []struct {
		name         string
		quest        *quest.Quest
		questID      func(q *quest.Quest) uuid.UUID
		adventurerID uuid.UUID
		updateErr    error
		wantErr      error
	}{
		{
			name:         "冒険者が存在しない",
			quest:        openQuest(t, "x", nil, testNow),
			adventurerID: uuid.Must(uuid.NewV7()),
			wantErr:      adventurer.ErrAdventurerNotFound,
		},
		{
			name:         "クエストが存在しない",
			quest:        openQuest(t, "x", nil, testNow),
			questID:      func(*quest.Quest) uuid.UUID { return uuid.Must(uuid.NewV7()) },
			adventurerID: a.ID(),
			wantErr:      quest.ErrQuestNotFound,
		},
		{
			name:         "期限切れ",
			quest:        openQuest(t, "x", &past, testNow.Add(-time.Hour)),
			adventurerID: a.ID(),
			wantErr:      quest.ErrQuestExpired,
		},
		{
			name:         "既に受注済",
			quest:        acceptedQuest(t, uuid.Must(uuid.NewV7())),
			adventurerID: a.ID(),
			wantErr:      quest.ErrQuestNotOpen,
		},
		{
			name:         "同時受注で競合",
			quest:        openQuest(t, "x", nil, testNow),
			adventurerID: a.ID(),
			updateErr:    quest.ErrConcurrentModification,
			wantErr:      quest.ErrQuestNotOpen,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeQuestRepo(tt.quest)
			repo.updateErr = tt.updateErr
			uc := NewAcceptQuest(repo, newFakeAdventurerRepo(a), FixedClock{T: testNow})
			id := tt.quest.ID()
			if tt.questID != nil {
				id = tt.questID(tt.quest)
			}

			_, err := uc.Execute(context.Background(), id, tt.adventurerID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
