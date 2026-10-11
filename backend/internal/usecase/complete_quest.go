package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// CompleteQuest は受注者本人が完了報告する（US4）。
type CompleteQuest struct {
	quests      quest.Repository
	adventurers adventurer.Repository
	clock       Clock
}

func NewCompleteQuest(quests quest.Repository, adventurers adventurer.Repository, clock Clock) *CompleteQuest {
	return &CompleteQuest{quests: quests, adventurers: adventurers, clock: clock}
}

// Execute の流れは AcceptQuest と同じ（冒険者の確認 → 取得 → Quest.Complete → 保存）。
//
// 受注済のクエストを変更できるのは受注者だけなので、競合は同じ冒険者による二重報告とみなし
// ErrQuestAlreadyCompleted を返す。
func (u *CompleteQuest) Execute(ctx context.Context, questID, adventurerID uuid.UUID) (QuestView, error) {
	a, err := u.adventurers.FindByID(ctx, adventurerID)
	if err != nil {
		return QuestView{}, err
	}
	q, err := u.quests.FindByID(ctx, questID)
	if err != nil {
		return QuestView{}, err
	}
	// 受注者本人かどうか・状態が受注済かどうかの判定は Quest.Complete の中にある
	if err := q.Complete(a.ID(), u.clock.Now()); err != nil {
		return QuestView{}, err
	}
	if err := u.quests.Update(ctx, q); err != nil {
		if errors.Is(err, quest.ErrConcurrentModification) {
			return QuestView{}, quest.ErrQuestAlreadyCompleted
		}
		return QuestView{}, err
	}
	return QuestView{Quest: q, Assignee: a}, nil
}
