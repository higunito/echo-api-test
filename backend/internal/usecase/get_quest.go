package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// GetQuest はクエストの詳細を返す（US2, FR-007）。
type GetQuest struct {
	quests      quest.Repository
	adventurers adventurer.Repository
}

func NewGetQuest(quests quest.Repository, adventurers adventurer.Repository) *GetQuest {
	return &GetQuest{quests: quests, adventurers: adventurers}
}

func (u *GetQuest) Execute(ctx context.Context, id uuid.UUID) (QuestView, error) {
	q, err := u.quests.FindByID(ctx, id)
	if err != nil {
		return QuestView{}, err // 見つからなければ quest.ErrQuestNotFound がそのまま返る
	}
	return withAssignee(ctx, u.adventurers, q)
}

// withAssignee はクエストに受注者を添えた QuestView を返す。
// メソッドではなく普通の関数にしているのは、どのユースケースからも使えるようにするため。
func withAssignee(ctx context.Context, adventurers adventurer.Repository, q *quest.Quest) (QuestView, error) {
	v := QuestView{Quest: q}
	if id := q.AssigneeID(); id != nil {
		a, err := adventurers.FindByID(ctx, *id)
		if err != nil {
			return QuestView{}, err
		}
		v.Assignee = a
	}
	return v, nil
}
