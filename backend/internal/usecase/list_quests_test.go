package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

func TestListQuests_Paging(t *testing.T) {
	tests := []struct {
		name           string
		count          int
		page           int
		wantItems      int
		wantPage       int
		wantTotalPages int
	}{
		{"0 件", 0, 0, 0, 1, 0},
		{"page 未指定は 1", 25, 0, 20, 1, 2},
		{"2 ページ目", 25, 2, 5, 2, 2},
		{"範囲外のページは空", 25, 3, 0, 3, 2},
		{"ちょうど 20 件", 20, 1, 20, 1, 1},
		{"21 件", 21, 1, 20, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeQuestRepo()
			for i := range tt.count {
				q := openQuest(t, "q", nil, testNow.Add(-time.Duration(i)*time.Minute))
				repo.quests[q.ID()] = q
			}
			uc := NewListQuests(repo, newFakeAdventurerRepo(), FixedClock{T: testNow})

			out, err := uc.Execute(context.Background(), ListQuestsInput{Page: tt.page})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Items) != tt.wantItems || out.Page != tt.wantPage ||
				out.TotalCount != tt.count || out.TotalPages != tt.wantTotalPages {
				t.Errorf("got items=%d page=%d total=%d pages=%d", len(out.Items), out.Page, out.TotalCount, out.TotalPages)
			}
		})
	}
}

func TestListQuests_InvalidPage(t *testing.T) {
	uc := NewListQuests(newFakeQuestRepo(), newFakeAdventurerRepo(), FixedClock{T: testNow})
	_, err := uc.Execute(context.Background(), ListQuestsInput{Page: -1})
	if !errors.Is(err, ErrInvalidPage) {
		t.Fatalf("err = %v, want ErrInvalidPage", err)
	}
}

func TestListQuests_PassesFilterAndClock(t *testing.T) {
	repo := newFakeQuestRepo()
	uc := NewListQuests(repo, newFakeAdventurerRepo(), FixedClock{T: testNow})
	status := quest.StatusOpen

	if _, err := uc.Execute(context.Background(), ListQuestsInput{Status: &status, ExcludeExpired: true, Page: 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listFilter.Status == nil || *repo.listFilter.Status != quest.StatusOpen || !repo.listFilter.ExcludeExpired {
		t.Errorf("filter = %+v", repo.listFilter)
	}
	if !repo.listNow.Equal(testNow) || repo.listPage != 2 {
		t.Errorf("now = %v, page = %d", repo.listNow, repo.listPage)
	}
}

func TestListQuests_ResolvesAssignee(t *testing.T) {
	a := newAdventurer("アリア")
	q := acceptedQuest(t, a.ID())
	uc := NewListQuests(newFakeQuestRepo(q), newFakeAdventurerRepo(a), FixedClock{T: testNow})

	out, err := uc.Execute(context.Background(), ListQuestsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].Assignee == nil || out.Items[0].Assignee.Name() != "アリア" {
		t.Errorf("assignee not resolved: %+v", out.Items)
	}
}
