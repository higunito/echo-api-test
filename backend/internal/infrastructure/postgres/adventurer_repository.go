package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/infrastructure/postgres/sqlcgen"
)

// AdventurerRepository は adventurer.Repository の Postgres 実装。
type AdventurerRepository struct {
	q *sqlcgen.Queries // sqlc が生成したクエリ関数の集まり
}

// コンパイル時のチェック: *AdventurerRepository が adventurer.Repository を満たしていなければビルドエラーになる。
// Go のインターフェースは暗黙的に実装されるため、メソッドの書き間違いに気づけるようこの 1 行を置くのが定番。
// `var _ 型 = 値` の _ は「使わない変数」。
var _ adventurer.Repository = (*AdventurerRepository)(nil)

func NewAdventurerRepository(pool *pgxpool.Pool) *AdventurerRepository {
	return &AdventurerRepository{q: sqlcgen.New(pool)}
}

func (r *AdventurerRepository) FindByID(ctx context.Context, id uuid.UUID) (*adventurer.Adventurer, error) {
	// GetAdventurer は db/queries/adventurers.sql の「-- name: GetAdventurer :one」から生成された関数
	row, err := r.q.GetAdventurer(ctx, id)
	// 1 行も見つからないと pgx は pgx.ErrNoRows を返すので、ドメインのエラーに読み替える。
	// （上の層が pgx を知らなくて済むように）
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, adventurer.ErrAdventurerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get adventurer: %w", err)
	}
	return adventurer.Reconstruct(row.ID, row.Name, row.CreatedAt), nil
}

func (r *AdventurerRepository) ListAll(ctx context.Context) ([]*adventurer.Adventurer, error) {
	rows, err := r.q.ListAdventurers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list adventurers: %w", err)
	}
	out := make([]*adventurer.Adventurer, 0, len(rows))
	for _, row := range rows {
		out = append(out, adventurer.Reconstruct(row.ID, row.Name, row.CreatedAt))
	}
	return out, nil
}
