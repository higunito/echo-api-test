package quest

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PageSize は一覧の 1 ページあたりの件数（FR-006a）。
const PageSize = 20

// Filter は一覧の絞り込み条件。
type Filter struct {
	Status         *Status // nil なら状態で絞り込まない（ポインタで「指定なし」を表す）
	ExcludeExpired bool    // true なら期限切れ（募集中かつ期限 <= now）を除く
}

// Repository はクエストの保存・取得のインターフェース。
//
// インターフェースを「使う側」の domain に定義し、「実装する側」は infrastructure/postgres に置く。
// これが依存性逆転（DIP）で、domain は DB の存在を知らずに済む。
// usecase のテストでは、このインターフェースを満たすメモリ上の偽物（usecase/fake_test.go）に差し替える。
//
// Go のインターフェースは「暗黙的に」実装される。Java の implements のような宣言は不要で、
// 同じシグネチャのメソッドをすべて持っていれば、その型はこのインターフェースを満たす。
//
// どのメソッドも第 1 引数に context.Context を取るのが Go の慣習。
// リクエストのキャンセルやタイムアウトを DB の呼び出しまで伝えるために使う。
type Repository interface {
	Create(ctx context.Context, q *Quest) error
	// FindByID は存在しなければ ErrQuestNotFound を返す。
	FindByID(ctx context.Context, id uuid.UUID) (*Quest, error)
	// Update は version が一致するときだけ保存し、成功時に q の version を進める。
	// 一致しなければ ErrConcurrentModification を返す。
	//
	// 例: A と B が同時に version=1 のクエストを読み、どちらも受注しようとする。
	// 先に保存した A で version は 2 になるので、B の「version=1 なら更新」は 0 件となり、競合として検出できる。
	Update(ctx context.Context, q *Quest) error
	// List は新しい順に page（1 始まり）の PageSize 件と、条件に合う総件数を返す。
	// 期限切れの判定に使う現在時刻 now は呼び出し側が渡す（テストで時刻を固定するため）。
	List(ctx context.Context, f Filter, now time.Time, page int) ([]*Quest, int, error)
}
