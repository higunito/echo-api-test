package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// CreateQuestInput はクエスト登録の入力。HTTP の JSON とは別の型にしておくと、
// usecase が HTTP の形式（JSON のキー名など）に引きずられない。
type CreateQuestInput struct {
	Title       string
	Description string
	Reward      int
	Deadline    *time.Time // 任意項目なので nil 可
}

// CreateQuest はギルドがクエストを登録する（US1）。
type CreateQuest struct {
	quests quest.Repository
	clock  Clock
}

func NewCreateQuest(quests quest.Repository, clock Clock) *CreateQuest {
	return &CreateQuest{quests: quests, clock: clock}
}

// Execute の流れ: ID を採番 → domain の NewQuest で検証しつつ作成 → 保存。
func (u *CreateQuest) Execute(ctx context.Context, in CreateQuestInput) (*quest.Quest, error) {
	// UUID v7 は先頭に時刻が入るので、作成順に並びやすい（インデックスの効率が良い）。
	id, err := uuid.NewV7()
	if err != nil {
		// %w でラップすると、元のエラーを残したまま「どこで失敗したか」の説明を足せる。
		// （errors.Is / errors.As で元のエラーを取り出せる。%v だと元のエラーが失われる）
		return nil, fmt.Errorf("generate quest id: %w", err)
	}
	q, err := quest.NewQuest(id, in.Title, in.Description, in.Reward, in.Deadline, u.clock.Now())
	if err != nil {
		// ドメインエラー（ErrInvalidTitle など）はそのまま返し、HTTP への変換は handler に任せる。
		return nil, err
	}
	if err := u.quests.Create(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}
