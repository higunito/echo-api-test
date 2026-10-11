package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

func TestCompleteQuest(t *testing.T) {
	a := newAdventurer("アリア")
	q := acceptedQuest(t, a.ID())
	uc := NewCompleteQuest(newFakeQuestRepo(q), newFakeAdventurerRepo(a), FixedClock{T: testNow})

	v, err := uc.Execute(context.Background(), q.ID(), a.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Quest.Status() != quest.StatusCompleted || !v.Quest.CompletedAt().Equal(testNow) || v.Assignee.ID() != a.ID() {
		t.Errorf("unexpected view: %+v", v.Quest)
	}
	if v.Quest.Version() != 2 {
		t.Errorf("version = %d, want 2 (Update must be called)", v.Quest.Version())
	}
}

func TestCompleteQuest_Errors(t *testing.T) {
	a := newAdventurer("アリア")
	b := newAdventurer("ボルグ")
	tests := []struct {
		name         string
		quest        *quest.Quest
		questID      *uuid.UUID
		adventurerID uuid.UUID
		updateErr    error
		wantErr      error
	}{
		{name: "冒険者が存在しない", quest: acceptedQuest(t, a.ID()), adventurerID: uuid.Must(uuid.NewV7()), wantErr: adventurer.ErrAdventurerNotFound},
		{name: "クエストが存在しない", quest: acceptedQuest(t, a.ID()), questID: new(uuid.Must(uuid.NewV7())), adventurerID: a.ID(), wantErr: quest.ErrQuestNotFound},
		{name: "受注者以外", quest: acceptedQuest(t, a.ID()), adventurerID: b.ID(), wantErr: quest.ErrNotAssignee},
		{name: "未受注", quest: openQuest(t, "x", nil, testNow), adventurerID: a.ID(), wantErr: quest.ErrQuestNotAccepted},
		{name: "同時に完了報告して競合", quest: acceptedQuest(t, a.ID()), adventurerID: a.ID(), updateErr: quest.ErrConcurrentModification, wantErr: quest.ErrQuestAlreadyCompleted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeQuestRepo(tt.quest)
			repo.updateErr = tt.updateErr
			uc := NewCompleteQuest(repo, newFakeAdventurerRepo(a, b), FixedClock{T: testNow})
			id := tt.quest.ID()
			if tt.questID != nil {
				id = *tt.questID
			}

			_, err := uc.Execute(context.Background(), id, tt.adventurerID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
