package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

var testNow = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func TestCreateQuest(t *testing.T) {
	repo := newFakeQuestRepo()
	uc := NewCreateQuest(repo, FixedClock{T: testNow})
	deadline := testNow.Add(24 * time.Hour)

	q, err := uc.Execute(context.Background(), CreateQuestInput{
		Title: "薬草採取", Description: "森で", Reward: 100, Deadline: &deadline,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Status() != quest.StatusOpen || !q.CreatedAt().Equal(testNow) {
		t.Errorf("unexpected quest: status=%s createdAt=%v", q.Status(), q.CreatedAt())
	}
	saved, err := repo.FindByID(context.Background(), q.ID())
	if err != nil {
		t.Fatalf("quest was not saved: %v", err)
	}
	if saved.Title() != "薬草採取" {
		t.Errorf("saved title = %q", saved.Title())
	}
}

func TestCreateQuest_DomainError(t *testing.T) {
	repo := newFakeQuestRepo()
	uc := NewCreateQuest(repo, FixedClock{T: testNow})

	_, err := uc.Execute(context.Background(), CreateQuestInput{Title: "", Reward: 100})
	if !errors.Is(err, quest.ErrInvalidTitle) {
		t.Fatalf("err = %v, want ErrInvalidTitle", err)
	}
	if len(repo.quests) != 0 {
		t.Error("invalid quest must not be saved")
	}
}
