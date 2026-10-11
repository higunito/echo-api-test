// usecase のテスト用の偽物（フェイク）をまとめたファイル。
//
// usecase は domain のインターフェース（quest.Repository など）にしか依存しないので、
// テストでは DB の代わりにメモリ上の map で動く偽物を渡せる。DB を起動せずに速くテストできる。
// ファイル名が _test.go なので、これらの型はテストのときだけ存在する。
package usecase

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// FixedClock は常に同じ時刻を返す Clock（clock.go のインターフェースを満たす）。
type FixedClock struct{ T time.Time }

func (c FixedClock) Now() time.Time { return c.T }

// fakeQuestRepo はメモリ上の quest.Repository。
// updateErr を設定すると Update を失敗させられるので、「同時受注で競合した」場合の分岐もテストできる。
type fakeQuestRepo struct {
	quests map[uuid.UUID]*quest.Quest
	// updateErr が設定されていれば Update はそれを返す。
	updateErr error
	// listArgs は最後の List 呼び出しの引数。
	listFilter quest.Filter
	listNow    time.Time
	listPage   int
}

func newFakeQuestRepo(qs ...*quest.Quest) *fakeQuestRepo {
	r := &fakeQuestRepo{quests: map[uuid.UUID]*quest.Quest{}}
	for _, q := range qs {
		r.quests[q.ID()] = q
	}
	return r
}

func (r *fakeQuestRepo) Create(_ context.Context, q *quest.Quest) error {
	r.quests[q.ID()] = q
	return nil
}

func (r *fakeQuestRepo) FindByID(_ context.Context, id uuid.UUID) (*quest.Quest, error) {
	q, ok := r.quests[id]
	if !ok {
		return nil, quest.ErrQuestNotFound
	}
	return q, nil
}

func (r *fakeQuestRepo) Update(_ context.Context, q *quest.Quest) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	q.IncrementVersion()
	r.quests[q.ID()] = q
	return nil
}

func (r *fakeQuestRepo) List(_ context.Context, f quest.Filter, now time.Time, page int) ([]*quest.Quest, int, error) {
	r.listFilter, r.listNow, r.listPage = f, now, page
	var all []*quest.Quest
	for _, q := range r.quests {
		if f.Status != nil && q.Status() != *f.Status {
			continue
		}
		if f.ExcludeExpired && q.IsExpired(now) {
			continue
		}
		all = append(all, q)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt().After(all[j].CreatedAt()) })
	start := (page - 1) * quest.PageSize
	if start >= len(all) {
		return []*quest.Quest{}, len(all), nil
	}
	end := min(start+quest.PageSize, len(all))
	return all[start:end], len(all), nil
}

type fakeAdventurerRepo struct {
	adventurers map[uuid.UUID]*adventurer.Adventurer
}

func newFakeAdventurerRepo(as ...*adventurer.Adventurer) *fakeAdventurerRepo {
	r := &fakeAdventurerRepo{adventurers: map[uuid.UUID]*adventurer.Adventurer{}}
	for _, a := range as {
		r.adventurers[a.ID()] = a
	}
	return r
}

func (r *fakeAdventurerRepo) FindByID(_ context.Context, id uuid.UUID) (*adventurer.Adventurer, error) {
	a, ok := r.adventurers[id]
	if !ok {
		return nil, adventurer.ErrAdventurerNotFound
	}
	return a, nil
}

func (r *fakeAdventurerRepo) ListAll(_ context.Context) ([]*adventurer.Adventurer, error) {
	var out []*adventurer.Adventurer
	for _, a := range r.adventurers {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}
