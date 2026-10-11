package usecase

import "time"

// Clock は現在時刻の取得元。テストで時刻を固定するために差し替える。
//
// コードの中で time.Now() を直接呼ぶと、「期限切れかどうか」のような時刻に依存する処理を
// テストで再現しにくい。インターフェースにしておけば、テストでは固定時刻を返す FixedClock を渡せる。
type Clock interface {
	Now() time.Time
}

// SystemClock は本番用の Clock。フィールドを持たない空の構造体（struct{}）で、メモリも使わない。
type SystemClock struct{}

// Now は現在時刻を UTC で返す。DB や API では UTC で扱い、日本時間への変換は画面側で行う（research R12）。
//
// レシーバが (SystemClock) と値になっているのは、フィールドがなく書き換えるものもないため。
// このメソッドを持つので、SystemClock は Clock インターフェースを（宣言なしで）満たす。
func (SystemClock) Now() time.Time { return time.Now().UTC() }
