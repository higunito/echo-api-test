// 先頭の //go:build integration は「ビルドタグ」。
// `go test -tags=integration` を付けたときだけこのファイルがコンパイルされる。
// DB が必要なテストを、普段の `go test ./...`（mise run test）から外すために使っている。
// 統合テストの実行: mise run test:integration（テスト用 DB へのマイグレーション適用も先に行う）
//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDatabaseURL = "postgres://guild:guild@localhost:5433/guild_test?sslmode=disable"

// setupDB はテスト用 DB に接続し、テーブルを空にしたプールを返す。
// テスト用 DB には事前に `mise run db:migrate:test` でスキーマを適用しておくこと。
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = defaultTestDatabaseURL
	}
	ctx := context.Background()
	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE quests, adventurers"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

// insertAdventurer はテスト用の冒険者を 1 人登録し、その ID を返す。
func insertAdventurer(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(context.Background(),
		"INSERT INTO adventurers (id, name, created_at) VALUES ($1, $2, $3)",
		id, name, time.Now().UTC()); err != nil {
		t.Fatalf("insert adventurer: %v", err)
	}
	return id
}
