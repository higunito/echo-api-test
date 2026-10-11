// Package postgres は domain のリポジトリインターフェースを PostgreSQL で実装する（infrastructure 層）。
//
// SQL は backend/db/queries/*.sql に書き、`mise run gen`（sqlc generate）で
// 型付きの Go コード（sqlcgen/ 配下。手編集禁止）を生成して使う。
// このパッケージの役割は、sqlc が生成した型（sqlcgen.Quest など）と domain の型（quest.Quest）の
// 相互変換と、DB のエラーをドメインエラーに読み替えること。
//
// ドライバは pgx/v5。database/sql を経由せず pgx を直接使うと、Postgres 固有の型を扱いやすく速い。
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool は接続プールを作成し、疎通を確認する。
//
// pgxpool.Pool は複数の DB 接続を使い回す仕組みで、複数の goroutine から同時に使っても安全。
// サーバ全体で 1 つ作り、各リポジトリで共有する（main.go 参照）。
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	// pgxpool.New は実際には接続しないので、Ping で接続できるか確認する。
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
