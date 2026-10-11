// Package adventurer は冒険者のドメインモデル。
//
// 本機能（001-quest-management）では参照だけで、登録・編集は別機能で扱う。
// 初期データは backend/db/seed.sql で投入する。
package adventurer

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrAdventurerNotFound は指定した冒険者がいないことを表す（センチネルエラー。quest/errors.go 参照）。
var ErrAdventurerNotFound = errors.New("adventurer: not found")

// Adventurer はクエストを受注・完了報告する人物。
// フィールドを非公開にし、ゲッター経由で読む作りは quest.Quest と同じ。
type Adventurer struct {
	id        uuid.UUID
	name      string
	createdAt time.Time
}

// Reconstruct は永続化された値から Adventurer を復元する。
func Reconstruct(id uuid.UUID, name string, createdAt time.Time) *Adventurer {
	return &Adventurer{id: id, name: name, createdAt: createdAt}
}

func (a *Adventurer) ID() uuid.UUID        { return a.id }
func (a *Adventurer) Name() string         { return a.name }
func (a *Adventurer) CreatedAt() time.Time { return a.createdAt }

// Repository は冒険者の取得のインターフェース。実装は infrastructure/postgres/adventurer_repository.go。
type Repository interface {
	// FindByID は存在しなければ ErrAdventurerNotFound を返す。
	FindByID(ctx context.Context, id uuid.UUID) (*Adventurer, error)
	// ListAll は全員を名前順で返す。
	ListAll(ctx context.Context) ([]*Adventurer, error)
}
