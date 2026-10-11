package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

func TestGetQuest(t *testing.T) {
	a := newAdventurer("アリア")
	q := acceptedQuest(t, a.ID())
	uc := NewGetQuest(newFakeQuestRepo(q), newFakeAdventurerRepo(a))

	v, err := uc.Execute(context.Background(), q.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Quest.ID() != q.ID() || v.Assignee == nil || v.Assignee.ID() != a.ID() {
		t.Errorf("unexpected view: %+v", v)
	}
}

func TestGetQuest_Open(t *testing.T) {
	q := openQuest(t, "募集中", nil, testNow)
	uc := NewGetQuest(newFakeQuestRepo(q), newFakeAdventurerRepo())

	v, err := uc.Execute(context.Background(), q.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Assignee != nil {
		t.Errorf("assignee = %v, want nil", v.Assignee)
	}
}

func TestGetQuest_NotFound(t *testing.T) {
	uc := NewGetQuest(newFakeQuestRepo(), newFakeAdventurerRepo())
	_, err := uc.Execute(context.Background(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, quest.ErrQuestNotFound) {
		t.Fatalf("err = %v, want ErrQuestNotFound", err)
	}
}
