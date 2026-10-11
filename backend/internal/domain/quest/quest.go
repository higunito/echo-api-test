package quest

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Quest はギルドが公開する依頼。DDD でいう「集約ルート」で、
// 状態遷移（募集中 → 受注済 → 完了）のルールはすべてこの型のメソッドに閉じ込める。
//
// フィールド名が小文字で始まっているのは「パッケージ外から直接触れない」ため。
// Go では名前の先頭が大文字なら公開（exported）、小文字なら非公開になる。
// 外からは NewQuest / Reconstruct で作り、Accept などのメソッド経由でしか状態を変えられないので、
// 不正な状態（例: 募集中なのに受注者がいる）を作れなくなる。
type Quest struct {
	id          uuid.UUID
	title       string
	description string
	reward      int
	// *time.Time（ポインタ）は「値がないこと」を nil で表すために使う。
	// time.Time のゼロ値（0001-01-01）と「未設定」を区別したいときの Go の定番の書き方。
	deadline    *time.Time
	status      Status
	assigneeID  *uuid.UUID // 未受注なら nil
	acceptedAt  *time.Time // 未受注なら nil
	completedAt *time.Time // 未完了なら nil
	createdAt   time.Time
	// version は楽観的ロック用。保存のたびに 1 増える（research R3, repository.go の Update 参照）。
	version int
}

// const ( ... ) で複数の定数をまとめて宣言できる。小文字なのでこのパッケージ内だけで使う。
const (
	maxTitleLength       = 100
	maxDescriptionLength = 2000
)

// NewQuest は「募集中」のクエストを作成する（FR-001〜FR-005）。
// タイトルは前後の空白を除去してから 1〜100 文字であることを検証する。
//
// Go には例外がないので、失敗しうる関数は「(結果, error)」の 2 つを返すのが慣習。
// 呼び出し側は `q, err := NewQuest(...)` で受け取り、`if err != nil` で失敗を判定する。
// 現在時刻 now を引数で受け取るのは、テストで時刻を固定できるようにするため（time.Now() を直接呼ばない）。
func NewQuest(id uuid.UUID, title, description string, reward int, deadline *time.Time, now time.Time) (*Quest, error) {
	title = strings.TrimSpace(title)
	// len(title) はバイト数を返すため、日本語（1 文字 3 バイト）だと文字数とずれる。
	// 文字数を数えるときは utf8.RuneCountInString を使う。
	// `if 初期化文; 条件 {` の形で、if の中だけで使う変数 n を宣言できる。
	if n := utf8.RuneCountInString(title); n < 1 || n > maxTitleLength {
		return nil, ErrInvalidTitle
	}
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return nil, ErrInvalidDescription
	}
	// DB の integer（32 ビット）に収まる範囲に制限する。
	// これがないと、保存時の int32 変換で大きな値が別の値に化けてしまう。
	if reward < 0 || reward > math.MaxInt32 {
		return nil, ErrInvalidReward
	}
	// deadline は nil の可能性があるので、先に nil チェックしてから中身（*deadline のメソッド）を使う。
	// `!deadline.After(now)` は「deadline <= now」と同じ意味（ちょうど現在時刻も過去扱い）。
	if deadline != nil && !deadline.After(now) {
		return nil, ErrDeadlineInPast
	}
	// &Quest{...} は構造体を作ってそのポインタを返す書き方。
	// 指定しなかったフィールド（assigneeID など）は型のゼロ値（ポインタなら nil）になる。
	return &Quest{
		id:          id,
		title:       title,
		description: description,
		reward:      reward,
		deadline:    deadline,
		status:      StatusOpen,
		createdAt:   now,
		version:     1,
	}, nil
}

// Reconstruct は永続化された値から Quest を復元する。検証は行わない。
// DB から読み出すとき（infrastructure 層）とテストで使う。
// NewQuest と分けているのは、「新規作成時のルール」（例: 期限は未来）を、
// 既に保存済みのデータ（期限が過ぎたクエストなど）に当てはめないため。
func Reconstruct(
	id uuid.UUID,
	title, description string, // 同じ型が続く引数は型をまとめて書ける
	reward int,
	deadline *time.Time,
	status Status,
	assigneeID *uuid.UUID,
	acceptedAt, completedAt *time.Time,
	createdAt time.Time,
	version int,
) *Quest {
	return &Quest{
		id:          id,
		title:       title,
		description: description,
		reward:      reward,
		deadline:    deadline,
		status:      status,
		assigneeID:  assigneeID,
		acceptedAt:  acceptedAt,
		completedAt: completedAt,
		createdAt:   createdAt,
		version:     version,
	}
}

// 以下はゲッター。Go では「GetID」ではなく「ID」のように Get を付けない名前が慣習。
// `func (q *Quest) ...` の (q *Quest) は「レシーバ」で、Quest のメソッドであることを表す。
// ポインタレシーバ（*Quest）にしておくと、Accept などで q のフィールドを書き換えられる。
// 1 つの型ではレシーバの種類（ポインタか値か）をそろえるのが慣習なので、ゲッターもポインタにしている。
func (q *Quest) ID() uuid.UUID           { return q.id }
func (q *Quest) Title() string           { return q.title }
func (q *Quest) Description() string     { return q.description }
func (q *Quest) Reward() int             { return q.reward }
func (q *Quest) Deadline() *time.Time    { return q.deadline }
func (q *Quest) Status() Status          { return q.status }
func (q *Quest) AssigneeID() *uuid.UUID  { return q.assigneeID }
func (q *Quest) AcceptedAt() *time.Time  { return q.acceptedAt }
func (q *Quest) CompletedAt() *time.Time { return q.completedAt }
func (q *Quest) CreatedAt() time.Time    { return q.createdAt }
func (q *Quest) Version() int            { return q.version }

// IsExpired は募集中のまま期限を過ぎているかを返す。保存される状態ではなく都度判定する（FR-007a）。
// 受注済・完了のクエストは期限を過ぎていても「期限切れ」にはしない。
func (q *Quest) IsExpired(now time.Time) bool {
	return q.status == StatusOpen && q.deadline != nil && !q.deadline.After(now)
}

// Accept は冒険者がクエストを受注する（FR-008）。募集中かつ期限内のときだけ成功する。
//
// エラーのときは何も変更せずに return する（途中まで書き換えた中途半端な状態を残さない）。
// このメソッドはメモリ上の Quest を変えるだけで、DB への保存は usecase が repository.Update で行う。
func (q *Quest) Accept(adventurerID uuid.UUID, now time.Time) error {
	if q.status != StatusOpen {
		return ErrQuestNotOpen
	}
	if q.IsExpired(now) {
		return ErrQuestExpired
	}
	q.status = StatusAccepted
	// &adventurerID は引数（このメソッドのローカル変数）のアドレス。
	// Go ではローカル変数のアドレスを返したり保存したりしても安全（必要ならヒープに移される）。
	q.assigneeID = &adventurerID
	q.acceptedAt = &now
	return nil
}

// IncrementVersion は永続化に成功したあとにリポジトリが呼ぶ（楽観的ロック。research R3）。
// リポジトリは別パッケージ（infrastructure/postgres）にあり、非公開フィールド version を直接触れないため、
// 公開メソッドとして用意している。
func (q *Quest) IncrementVersion() {
	q.version++
}

// Complete は受注者本人が完了報告する（FR-010）。期限を過ぎていても報告できる。
func (q *Quest) Complete(adventurerID uuid.UUID, now time.Time) error {
	// Go の switch は case ごとに自動で break する（C や JS のように break を書かなくてよい）。
	// どの case にも当てはまらなければ（= StatusAccepted なら）switch を抜けて次へ進む。
	switch q.status {
	case StatusOpen:
		return ErrQuestNotAccepted
	case StatusCompleted:
		return ErrQuestAlreadyCompleted
	}
	// *q.assigneeID でポインタの指す値を取り出して比較する。nil のまま * すると panic するので先に nil チェック。
	// （|| は左が true なら右を評価しないので、この順番なら安全）
	if q.assigneeID == nil || *q.assigneeID != adventurerID {
		return ErrNotAssignee
	}
	q.status = StatusCompleted
	q.completedAt = &now
	return nil
}
