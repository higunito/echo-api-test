// テストファイルは _test.go で終わる名前にする。`go test ./...` のときだけコンパイルされ、本番のバイナリには入らない。
// package quest（_test を付けない）にしているので、同じパッケージの非公開の識別子も参照できる。
package quest

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// テストで使う固定の「現在時刻」。time.Now() を使わないので、何度実行しても結果が変わらない。
var now = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

// ptr は値のポインタを返す小さなヘルパー。[T any] はジェネリクス（型パラメータ）で、どの型にも使える。
// 例: ptr(now) で *time.Time を作る（&now と書けない式の値のポインタが欲しいときに便利）。
func ptr[T any](v T) *T { return &v }

// TestXxx(t *testing.T) という名前の関数が、go test でテストとして実行される。
//
// 「テーブル駆動テスト」は Go の定番の書き方。入力と期待値の組をスライス（tests）に並べ、
// ループで 1 件ずつ t.Run(名前, ...) のサブテストとして実行する。
// ケースを足すときは表に 1 行追加するだけでよく、失敗したケースは名前付きで表示される。
// 特定のケースだけ実行: go test ./internal/domain/quest -run 'TestNewQuest/タイトル空'
func TestNewQuest(t *testing.T) {
	future := now.Add(time.Hour)
	tests := []struct {
		name        string
		title       string
		description string
		reward      int
		deadline    *time.Time
		wantErr     error
		wantTitle   string
	}{
		{name: "正常", title: "薬草採取", description: "森で薬草を集める", reward: 100, deadline: &future, wantTitle: "薬草採取"},
		{name: "期限なし", title: "薬草採取", reward: 100, deadline: nil, wantTitle: "薬草採取"},
		{name: "報酬 0 は可", title: "x", reward: 0, wantTitle: "x"},
		{name: "タイトルの前後の空白を除去", title: "  薬草採取 ", reward: 1, wantTitle: "薬草採取"},
		{name: "タイトル 100 文字は可", title: strings.Repeat("あ", 100), reward: 1, wantTitle: strings.Repeat("あ", 100)},
		{name: "タイトル空", title: "", reward: 1, wantErr: ErrInvalidTitle},
		{name: "タイトル空白のみ", title: "   ", reward: 1, wantErr: ErrInvalidTitle},
		{name: "タイトル 101 文字", title: strings.Repeat("あ", 101), reward: 1, wantErr: ErrInvalidTitle},
		{name: "説明 2000 文字は可", title: "x", description: strings.Repeat("あ", 2000), reward: 1, wantTitle: "x"},
		{name: "説明 2001 文字", title: "x", description: strings.Repeat("あ", 2001), reward: 1, wantErr: ErrInvalidDescription},
		{name: "報酬が負", title: "x", reward: -1, wantErr: ErrInvalidReward},
		{name: "報酬が上限", title: "x", reward: math.MaxInt32, wantTitle: "x"},
		{name: "報酬が上限超過", title: "x", reward: math.MaxInt32 + 1, wantErr: ErrInvalidReward},
		{name: "期限が現在と同時刻", title: "x", reward: 1, deadline: ptr(now), wantErr: ErrDeadlineInPast},
		{name: "期限が過去", title: "x", reward: 1, deadline: ptr(now.Add(-time.Second)), wantErr: ErrDeadlineInPast},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.Must(uuid.NewV7())
			q, err := NewQuest(id, tt.title, tt.description, tt.reward, tt.deadline, now)
			if tt.wantErr != nil {
				// t.Fatalf は失敗を記録して、このテスト（サブテスト）をその場で終了する。
				// t.Errorf は失敗を記録するが続行する（複数の不一致をまとめて見たいときに使う）。
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.ID() != id || q.Title() != tt.wantTitle || q.Description() != tt.description || q.Reward() != tt.reward {
				t.Errorf("unexpected fields: %+v", q)
			}
			if q.Status() != StatusOpen {
				t.Errorf("status = %s, want open", q.Status())
			}
			if q.Version() != 1 {
				t.Errorf("version = %d, want 1", q.Version())
			}
			if !q.CreatedAt().Equal(now) {
				t.Errorf("createdAt = %v, want %v", q.CreatedAt(), now)
			}
			if q.AssigneeID() != nil || q.AcceptedAt() != nil || q.CompletedAt() != nil {
				t.Error("assignee / acceptedAt / completedAt must be nil")
			}
			if (tt.deadline == nil) != (q.Deadline() == nil) {
				t.Errorf("deadline = %v, want %v", q.Deadline(), tt.deadline)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	assignee := uuid.Must(uuid.NewV7())
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	reconstruct := func(status Status, deadline *time.Time) *Quest {
		var a *uuid.UUID
		var acceptedAt, completedAt *time.Time
		if status != StatusOpen {
			a, acceptedAt = &assignee, &past
		}
		if status == StatusCompleted {
			completedAt = &past
		}
		return Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, deadline, status, a, acceptedAt, completedAt, past.Add(-time.Hour), 1)
	}
	tests := []struct {
		name string
		q    *Quest
		want bool
	}{
		{"募集中・期限切れ", reconstruct(StatusOpen, &past), true},
		{"募集中・期限ちょうど", reconstruct(StatusOpen, ptr(now)), true},
		{"募集中・期限内", reconstruct(StatusOpen, &future), false},
		{"募集中・期限なし", reconstruct(StatusOpen, nil), false},
		{"受注済・期限切れ", reconstruct(StatusAccepted, &past), false},
		{"完了・期限切れ", reconstruct(StatusCompleted, &past), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.IsExpired(now); got != tt.want {
				t.Errorf("IsExpired = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccept(t *testing.T) {
	adventurerID := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())
	created := now.Add(-2 * time.Hour)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	tests := []struct {
		name    string
		q       *Quest
		wantErr error
	}{
		{"募集中・期限内", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, &future, StatusOpen, nil, nil, nil, created, 1), nil},
		{"募集中・期限なし", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, nil, StatusOpen, nil, nil, nil, created, 1), nil},
		{"募集中・期限切れ", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, &past, StatusOpen, nil, nil, nil, created, 1), ErrQuestExpired},
		{"受注済", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, nil, StatusAccepted, &other, &past, nil, created, 2), ErrQuestNotOpen},
		{"完了", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, nil, StatusCompleted, &other, &past, &past, created, 3), ErrQuestNotOpen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// *tt.q で構造体の中身をコピーしておき、エラー時に何も変わっていないことを == で確かめる
			before := *tt.q
			err := tt.q.Accept(adventurerID, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if *tt.q != before {
					t.Error("quest must not change on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.q.Status() != StatusAccepted || tt.q.AssigneeID() == nil || *tt.q.AssigneeID() != adventurerID ||
				tt.q.AcceptedAt() == nil || !tt.q.AcceptedAt().Equal(now) || tt.q.CompletedAt() != nil {
				t.Errorf("unexpected state: %+v", tt.q)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	assignee := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())
	created := now.Add(-3 * time.Hour)
	acceptedAt := now.Add(-2 * time.Hour)
	past := now.Add(-time.Hour)
	accepted := func(deadline *time.Time) *Quest {
		return Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, deadline, StatusAccepted, &assignee, &acceptedAt, nil, created, 2)
	}
	tests := []struct {
		name         string
		q            *Quest
		adventurerID uuid.UUID
		wantErr      error
	}{
		{"受注者本人", accepted(nil), assignee, nil},
		{"期限を過ぎた受注済でも可", accepted(&past), assignee, nil},
		{"受注者以外", accepted(nil), other, ErrNotAssignee},
		{"募集中", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, nil, StatusOpen, nil, nil, nil, created, 1), assignee, ErrQuestNotAccepted},
		{"完了済", Reconstruct(uuid.Must(uuid.NewV7()), "x", "", 1, nil, StatusCompleted, &assignee, &acceptedAt, &past, created, 3), assignee, ErrQuestAlreadyCompleted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := *tt.q
			err := tt.q.Complete(tt.adventurerID, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if *tt.q != before {
					t.Error("quest must not change on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.q.Status() != StatusCompleted || tt.q.CompletedAt() == nil || !tt.q.CompletedAt().Equal(now) ||
				*tt.q.AssigneeID() != assignee || !tt.q.AcceptedAt().Equal(acceptedAt) {
				t.Errorf("unexpected state: %+v", tt.q)
			}
		})
	}
}
