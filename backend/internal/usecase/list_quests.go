package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// ErrInvalidPage はページ番号が不正（負の数）であることを表す。handler で 400 VALIDATION_ERROR に変換される。
var ErrInvalidPage = errors.New("usecase: page must be 1 or greater")

type ListQuestsInput struct {
	Status         *quest.Status // nil なら状態で絞り込まない
	ExcludeExpired bool
	// Page は 1 始まり。0 は未指定として 1 ページ目を返す。
	// （int のゼロ値は 0 なので、ListQuestsInput{} と書くだけで「未指定」になる）
	Page int
}

type ListQuestsOutput struct {
	Items      []QuestView
	Page       int
	TotalCount int
	TotalPages int
}

// ListQuests はクエスト一覧を返す（US2, FR-006, FR-006a, FR-007b）。
type ListQuests struct {
	quests      quest.Repository
	adventurers adventurer.Repository
	clock       Clock
}

func NewListQuests(quests quest.Repository, adventurers adventurer.Repository, clock Clock) *ListQuests {
	return &ListQuests{quests: quests, adventurers: adventurers, clock: clock}
}

func (u *ListQuests) Execute(ctx context.Context, in ListQuestsInput) (ListQuestsOutput, error) {
	page := in.Page
	if page == 0 {
		page = 1
	}
	if page < 1 {
		// 構造体を返す関数でエラーのときは、ゼロ値（ListQuestsOutput{}）とエラーを返すのが慣習。
		return ListQuestsOutput{}, ErrInvalidPage
	}
	qs, total, err := u.quests.List(ctx, quest.Filter{Status: in.Status, ExcludeExpired: in.ExcludeExpired}, u.clock.Now(), page)
	if err != nil {
		return ListQuestsOutput{}, err
	}
	assignees, err := u.assignees(ctx, qs)
	if err != nil {
		return ListQuestsOutput{}, err
	}
	// make([]T, 0, n) は「長さ 0・容量 n」のスライスを作る。
	// 最終的な件数が分かっているときに容量を先に確保すると、append のたびの再確保が起きない。
	// また nil スライスと違い、JSON にすると null ではなく [] になる。
	items := make([]QuestView, 0, len(qs))
	for _, q := range qs { // range はインデックスと要素を返す。インデックスは不要なので _ で捨てる
		v := QuestView{Quest: q}
		if id := q.AssigneeID(); id != nil {
			v.Assignee = assignees[*id]
		}
		items = append(items, v)
	}
	return ListQuestsOutput{
		Items:      items,
		Page:       page,
		TotalCount: total,
		// 整数の割り算は切り捨てなので、(a + b - 1) / b で切り上げを計算する（例: 21 件 → 2 ページ）。
		TotalPages: (total + quest.PageSize - 1) / quest.PageSize,
	}, nil
}

// assignees はページ内の受注者を ID ごとに 1 回だけ取得する（最大 20 件）。
//
// map[キーの型]値の型 は Go の連想配列。`out[*id]` のように引き、
// `v, ok := out[key]` の形で「キーが存在するか」も同時に調べられる。
func (u *ListQuests) assignees(ctx context.Context, qs []*quest.Quest) (map[uuid.UUID]*adventurer.Adventurer, error) {
	out := map[uuid.UUID]*adventurer.Adventurer{}
	for _, q := range qs {
		id := q.AssigneeID()
		if id == nil {
			continue
		}
		if _, ok := out[*id]; ok {
			continue // 取得済みの受注者は再取得しない
		}
		a, err := u.adventurers.FindByID(ctx, *id)
		if err != nil {
			return nil, err
		}
		out[*id] = a
	}
	return out, nil
}
