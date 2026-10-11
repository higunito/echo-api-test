package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// AcceptQuest は冒険者がクエストを受注する（US3）。
type AcceptQuest struct {
	quests      quest.Repository
	adventurers adventurer.Repository
	clock       Clock
}

func NewAcceptQuest(quests quest.Repository, adventurers adventurer.Repository, clock Clock) *AcceptQuest {
	return &AcceptQuest{quests: quests, adventurers: adventurers, clock: clock}
}

// Execute の流れ:
//  1. 冒険者が存在するか確認する（認証がないため、存在しない ID で受注されるのを防ぐ）
//  2. クエストを DB から読む（この時点の version を覚えている）
//  3. Quest.Accept で受注できるか判定し、メモリ上の状態を変える（ルールは domain 側）
//  4. Update で保存する。読んでから保存するまでに他の人が受注していたら、version が変わっていて競合になる
//
// 同時受注で競合した場合は「既に受注されている」として ErrQuestNotOpen を返す（FR-009）。
func (u *AcceptQuest) Execute(ctx context.Context, questID, adventurerID uuid.UUID) (QuestView, error) {
	a, err := u.adventurers.FindByID(ctx, adventurerID)
	if err != nil {
		return QuestView{}, err
	}
	q, err := u.quests.FindByID(ctx, questID)
	if err != nil {
		return QuestView{}, err
	}
	if err := q.Accept(a.ID(), u.clock.Now()); err != nil {
		return QuestView{}, err
	}
	if err := u.quests.Update(ctx, q); err != nil {
		// errors.Is はラップされたエラーの中まで比較する。
		// 競合は利用者から見れば「先に誰かが受注した」と同じなので、分かりやすいエラーに読み替える。
		if errors.Is(err, quest.ErrConcurrentModification) {
			return QuestView{}, quest.ErrQuestNotOpen
		}
		return QuestView{}, err
	}
	return QuestView{Quest: q, Assignee: a}, nil
}
