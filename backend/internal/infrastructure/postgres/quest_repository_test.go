//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

var baseTime = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func mustNewQuest(t *testing.T, title string, deadline *time.Time, createdAt time.Time) *quest.Quest {
	t.Helper()
	q, err := quest.NewQuest(uuid.Must(uuid.NewV7()), title, "説明", 100, deadline, createdAt)
	if err != nil {
		t.Fatalf("NewQuest: %v", err)
	}
	return q
}

func TestQuestRepository_CreateAndFind(t *testing.T) {
	pool := setupDB(t)
	repo := NewQuestRepository(pool)
	ctx := context.Background()
	deadline := baseTime.Add(48 * time.Hour)
	q := mustNewQuest(t, "薬草採取", &deadline, baseTime)

	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := repo.FindByID(ctx, q.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID() != q.ID() || got.Title() != "薬草採取" || got.Description() != "説明" || got.Reward() != 100 ||
		got.Status() != quest.StatusOpen || got.Version() != 1 || !got.CreatedAt().Equal(baseTime) ||
		got.Deadline() == nil || !got.Deadline().Equal(deadline) ||
		got.AssigneeID() != nil || got.AcceptedAt() != nil || got.CompletedAt() != nil {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestQuestRepository_FindByID_NotFound(t *testing.T) {
	pool := setupDB(t)
	repo := NewQuestRepository(pool)

	_, err := repo.FindByID(context.Background(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, quest.ErrQuestNotFound) {
		t.Fatalf("err = %v, want ErrQuestNotFound", err)
	}
}

// seedQuests は createdAt が 1 分ずつ古くなる募集中クエストを n 件登録し、新しい順に返す。
func seedQuests(t *testing.T, repo *QuestRepository, n int) []*quest.Quest {
	t.Helper()
	out := make([]*quest.Quest, 0, n)
	for i := range n {
		q := mustNewQuest(t, "q", nil, baseTime.Add(-time.Duration(i)*time.Minute))
		if err := repo.Create(context.Background(), q); err != nil {
			t.Fatalf("Create: %v", err)
		}
		out = append(out, q)
	}
	return out
}

func TestQuestRepository_List_Paging(t *testing.T) {
	pool := setupDB(t)
	repo := NewQuestRepository(pool)
	ctx := context.Background()
	seeded := seedQuests(t, repo, 25)

	for _, tt := range []struct{ page, want int }{{1, 20}, {2, 5}, {3, 0}} {
		got, total, err := repo.List(ctx, quest.Filter{}, baseTime, tt.page)
		if err != nil {
			t.Fatalf("List page %d: %v", tt.page, err)
		}
		if len(got) != tt.want || total != 25 {
			t.Errorf("page %d: len = %d, total = %d", tt.page, len(got), total)
		}
		for i, q := range got {
			if want := seeded[(tt.page-1)*quest.PageSize+i]; q.ID() != want.ID() {
				t.Errorf("page %d [%d]: order mismatch", tt.page, i)
			}
		}
	}
}

func TestQuestRepository_List_Filters(t *testing.T) {
	pool := setupDB(t)
	repo := NewQuestRepository(pool)
	ctx := context.Background()
	adventurerID := insertAdventurer(t, pool, "アリア")

	future := baseTime.Add(time.Hour)
	past := baseTime.Add(-time.Hour)
	created := baseTime.Add(-2 * time.Hour)
	open := mustNewQuest(t, "募集中", &future, created)
	noDeadline := mustNewQuest(t, "期限なし", nil, created)
	// 期限切れのクエストは NewQuest では作れないため、作成時刻を期限より前にして作る
	expired := mustNewQuest(t, "期限切れ", &past, created)
	acceptedExpired := quest.Reconstruct(uuid.Must(uuid.NewV7()), "受注済・期限切れ", "", 1, &past, quest.StatusAccepted,
		&adventurerID, &created, nil, created, 1)
	for _, q := range []*quest.Quest{open, noDeadline, expired, acceptedExpired} {
		if err := repo.Create(ctx, q); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	statusOpen := quest.StatusOpen
	statusAccepted := quest.StatusAccepted

	tests := []struct {
		name   string
		filter quest.Filter
		want   []*quest.Quest
	}{
		{"絞り込みなし", quest.Filter{}, []*quest.Quest{open, noDeadline, expired, acceptedExpired}},
		{"募集中", quest.Filter{Status: &statusOpen}, []*quest.Quest{open, noDeadline, expired}},
		{"受注済", quest.Filter{Status: &statusAccepted}, []*quest.Quest{acceptedExpired}},
		{"期限切れを除く", quest.Filter{ExcludeExpired: true}, []*quest.Quest{open, noDeadline, acceptedExpired}},
		{"募集中かつ期限切れを除く", quest.Filter{Status: &statusOpen, ExcludeExpired: true}, []*quest.Quest{open, noDeadline}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := repo.List(ctx, tt.filter, baseTime, 1)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if total != len(tt.want) || len(got) != len(tt.want) {
				t.Fatalf("len = %d, total = %d, want %d", len(got), total, len(tt.want))
			}
			wantIDs := map[uuid.UUID]bool{}
			for _, q := range tt.want {
				wantIDs[q.ID()] = true
			}
			for _, q := range got {
				if !wantIDs[q.ID()] {
					t.Errorf("unexpected quest %q", q.Title())
				}
			}
		})
	}
}

func TestQuestRepository_Update(t *testing.T) {
	pool := setupDB(t)
	repo := NewQuestRepository(pool)
	ctx := context.Background()
	adventurerID := insertAdventurer(t, pool, "アリア")
	q := mustNewQuest(t, "薬草採取", nil, baseTime)
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	stale, _ := repo.FindByID(ctx, q.ID())
	if err := q.Accept(adventurerID, baseTime.Add(time.Minute)); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if err := repo.Update(ctx, q); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if q.Version() != 2 {
		t.Errorf("version = %d, want 2", q.Version())
	}
	got, _ := repo.FindByID(ctx, q.ID())
	if got.Status() != quest.StatusAccepted || got.AssigneeID() == nil || *got.AssigneeID() != adventurerID || got.Version() != 2 {
		t.Errorf("not persisted: %+v", got)
	}

	// 古い version のまま保存しようとすると競合になる
	if err := stale.Accept(adventurerID, baseTime.Add(time.Minute)); err != nil {
		t.Fatalf("Accept stale: %v", err)
	}
	if err := repo.Update(ctx, stale); !errors.Is(err, quest.ErrConcurrentModification) {
		t.Fatalf("err = %v, want ErrConcurrentModification", err)
	}
}

// TestQuestRepository_ConcurrentAccept は同時受注で二重受注が起きないことを確認する（SC-004）。
func TestQuestRepository_ConcurrentAccept(t *testing.T) {
	pool := setupDB(t)
	repo := NewQuestRepository(pool)
	ctx := context.Background()
	q := mustNewQuest(t, "人気のクエスト", nil, baseTime)
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	const n = 100
	adventurers := make([]uuid.UUID, n)
	for i := range n {
		adventurers[i] = insertAdventurer(t, pool, "冒険者")
	}

	type result struct {
		adventurerID uuid.UUID
		err          error
	}
	// 100 個の goroutine で同時に「読む → Accept → 保存」を行う。
	// start チャネルを close するまで全員を待たせ、一斉にスタートさせて競合を起こしやすくする。
	// 結果は results チャネルに集める（goroutine から直接スライスに書くとデータ競合になるため）。
	results := make(chan result, n)
	start := make(chan struct{})
	for _, id := range adventurers {
		go func() {
			<-start
			found, err := repo.FindByID(ctx, q.ID())
			if err == nil {
				err = found.Accept(id, baseTime.Add(time.Minute))
			}
			if err == nil {
				err = repo.Update(ctx, found)
			}
			results <- result{id, err}
		}()
	}
	close(start)

	var winners []uuid.UUID
	for range n {
		r := <-results
		switch {
		case r.err == nil:
			winners = append(winners, r.adventurerID)
		case errors.Is(r.err, quest.ErrConcurrentModification), errors.Is(r.err, quest.ErrQuestNotOpen):
		default:
			t.Errorf("unexpected error: %v", r.err)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("successful accepts = %d, want exactly 1", len(winners))
	}
	got, _ := repo.FindByID(ctx, q.ID())
	if got.AssigneeID() == nil || *got.AssigneeID() != winners[0] {
		t.Errorf("assignee = %v, want %v", got.AssigneeID(), winners[0])
	}
}
