package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
	"github.com/higunito/echo-api-test/backend/internal/infrastructure/postgres/sqlcgen"
)

// QuestRepository は quest.Repository の Postgres 実装。
// SQL 本体は backend/db/queries/quests.sql を参照。
type QuestRepository struct {
	q *sqlcgen.Queries
}

// インターフェースを満たしているかのコンパイル時チェック（adventurer_repository.go 参照）
var _ quest.Repository = (*QuestRepository)(nil)

func NewQuestRepository(pool *pgxpool.Pool) *QuestRepository {
	return &QuestRepository{q: sqlcgen.New(pool)}
}

// Create は新しいクエストを INSERT する。
// sqlc はクエリのパラメータを XxxParams 構造体として生成するので、domain の Quest から値を詰め替えて渡す。
func (r *QuestRepository) Create(ctx context.Context, q *quest.Quest) error {
	err := r.q.CreateQuest(ctx, sqlcgen.CreateQuestParams{
		ID:          q.ID(),
		Title:       q.Title(),
		Description: q.Description(),
		// DB の integer は 32 ビットなので int32 に変換する。範囲は domain（NewQuest）で検証済み。
		Reward:      int32(q.Reward()),
		Deadline:    q.Deadline(), // *time.Time の nil は SQL の NULL になる（sqlc.yaml の override 設定）
		Status:      string(q.Status()),
		AssigneeID:  toNullUUID(q.AssigneeID()),
		AcceptedAt:  q.AcceptedAt(),
		CompletedAt: q.CompletedAt(),
		CreatedAt:   q.CreatedAt(),
		Version:     int32(q.Version()),
	})
	if err != nil {
		return fmt.Errorf("create quest: %w", err)
	}
	return nil
}

// FindByID は 1 件取得する。見つからなければ quest.ErrQuestNotFound。
func (r *QuestRepository) FindByID(ctx context.Context, id uuid.UUID) (*quest.Quest, error) {
	row, err := r.q.GetQuest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, quest.ErrQuestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get quest: %w", err)
	}
	return toDomainQuest(row), nil
}

// Update は状態の変更を保存する。楽観的ロックで同時更新を検出する（research R3）。
//
// SQL は「UPDATE ... WHERE id = $id AND version = $version」。
// 読み込んだあとに他の誰かが更新していれば version が進んでいるので、更新件数が 0 になる。
// 行ロックを取り続けずに済み、ルールの判定（Quest.Accept など）は Go 側に残せるのが利点。
func (r *QuestRepository) Update(ctx context.Context, q *quest.Quest) error {
	// :execrows で生成した関数は「更新した行数」を返す
	n, err := r.q.UpdateQuest(ctx, sqlcgen.UpdateQuestParams{
		Status:      string(q.Status()),
		AssigneeID:  toNullUUID(q.AssigneeID()),
		AcceptedAt:  q.AcceptedAt(),
		CompletedAt: q.CompletedAt(),
		ID:          q.ID(),
		Version:     int32(q.Version()),
	})
	if err != nil {
		return fmt.Errorf("update quest: %w", err)
	}
	if n == 0 {
		return quest.ErrConcurrentModification
	}
	// DB 側の version は SQL で +1 済みなので、メモリ上の Quest もそろえる
	q.IncrementVersion()
	return nil
}

// List は一覧の 1 ページ分と総件数を返す（FR-006a, FR-007b）。
// 「総件数」と「ページの中身」は別のクエリ（CountQuests / ListQuests）で、同じ絞り込み条件を使う。
func (r *QuestRepository) List(ctx context.Context, f quest.Filter, now time.Time, page int) ([]*quest.Quest, int, error) {
	// pgtype.Text は NULL になりうる text を表す pgx の型。Valid: false なら NULL。
	// SQL 側は「status が NULL なら絞り込まない」と書いてある（quests.sql の sqlc.narg('status')）。
	var status pgtype.Text // ゼロ値は Valid: false（= NULL）
	if f.Status != nil {
		status = pgtype.Text{String: string(*f.Status), Valid: true}
	}
	total, err := r.q.CountQuests(ctx, sqlcgen.CountQuestsParams{
		Status:         status,
		ExcludeExpired: f.ExcludeExpired,
		Now:            now,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count quests: %w", err)
	}
	rows, err := r.q.ListQuests(ctx, sqlcgen.ListQuestsParams{
		Status:         status,
		ExcludeExpired: f.ExcludeExpired,
		Now:            now,
		PageLimit:      quest.PageSize,
		// 2 ページ目なら 20 件飛ばす。範囲外のページは 0 件が返るだけでエラーにはならない。
		PageOffset: int32((page - 1) * quest.PageSize),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list quests: %w", err)
	}
	out := make([]*quest.Quest, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainQuest(row))
	}
	return out, int(total), nil
}

// toDomainQuest は sqlc の行（sqlcgen.Quest）を domain の Quest に変換する。
func toDomainQuest(row sqlcgen.Quest) *quest.Quest {
	return quest.Reconstruct(
		row.ID,
		row.Title,
		row.Description,
		int(row.Reward),
		utcPtr(row.Deadline),
		quest.Status(row.Status),
		fromNullUUID(row.AssigneeID),
		utcPtr(row.AcceptedAt),
		utcPtr(row.CompletedAt),
		row.CreatedAt.UTC(),
		int(row.Version),
	)
}

// toNullUUID / fromNullUUID は「nil 可の *uuid.UUID」と「NULL 可の uuid.NullUUID」を変換する。
// sqlc は NULL 可の uuid 列を uuid.NullUUID（UUID と Valid の組）として生成する（sqlc.yaml の override）。
func toNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{} // Valid: false = NULL
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func fromNullUUID(n uuid.NullUUID) *uuid.UUID {
	if !n.Valid {
		return nil
	}
	return &n.UUID
}

// utcPtr は時刻を UTC にそろえる。pgx は timestamptz をローカルタイムゾーン（日本時間）で返すため、
// UTC に統一しておくと、API の出力やテストでの比較がぶれない。
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
