package quest

import "fmt"

// Status はクエストの状態。遷移は open → accepted → completed のみ（FR-011）。
//
// `type Status string` は string を元にした新しい型（defined type）を作る書き方。
// 中身は文字列だが、ただの string とは別の型として扱われるので、
// 「状態を渡すべき引数に、うっかり別の文字列を渡す」ミスをコンパイル時に防げる。
type Status string

// Go には enum 構文がないため、型付きの定数で列挙値を表すのが定番。
const (
	StatusOpen      Status = "open"      // 募集中
	StatusAccepted  Status = "accepted"  // 受注済
	StatusCompleted Status = "completed" // 完了
)

// ParseStatus は文字列を Status に変換する。未知の値はエラー。
// HTTP のクエリパラメータなど、外から来た文字列を Status にするときに使う。
func ParseStatus(s string) (Status, error) {
	// `switch 初期化文; 対象 {` で、switch の中だけで使う変数 st を作れる。
	// Status(s) は string → Status への型変換。
	switch st := Status(s); st {
	case StatusOpen, StatusAccepted, StatusCompleted: // case にはカンマ区切りで複数の値を書ける
		return st, nil
	default:
		// fmt.Errorf は書式付きでエラーを作る。%q は値をクォート付きで出力する（"foo" のように）。
		return "", fmt.Errorf("unknown quest status: %q", s)
	}
}
