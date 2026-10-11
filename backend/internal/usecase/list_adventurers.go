package usecase

import (
	"context"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
)

// ListAdventurers は冒険者の一覧を返す（画面で操作する冒険者を選ぶため。FR-013）。
// 処理はリポジトリを呼ぶだけだが、handler がリポジトリを直接触らないよう usecase を挟む。
type ListAdventurers struct {
	adventurers adventurer.Repository
}

func NewListAdventurers(adventurers adventurer.Repository) *ListAdventurers {
	return &ListAdventurers{adventurers: adventurers}
}

func (u *ListAdventurers) Execute(ctx context.Context) ([]*adventurer.Adventurer, error) {
	return u.adventurers.ListAll(ctx)
}
