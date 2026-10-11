// handler のテスト。usecase を偽物（fakeCreator など）にして、HTTP の入出力だけを確かめる。
//
// net/http/httptest を使うと、実際にサーバを起動せずにリクエストを処理できる:
//   - httptest.NewRequest でリクエストを作る
//   - httptest.NewRecorder がレスポンスを記録する（rec.Code でステータス、rec.Body で本文）
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
	"github.com/higunito/echo-api-test/backend/internal/usecase"
)

var handlerNow = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type fakeCreator struct {
	got usecase.CreateQuestInput
	err error
}

func (f *fakeCreator) Execute(_ context.Context, in usecase.CreateQuestInput) (*quest.Quest, error) {
	f.got = in
	if f.err != nil {
		return nil, f.err
	}
	return quest.NewQuest(uuid.Must(uuid.NewV7()), in.Title, in.Description, in.Reward, in.Deadline, handlerNow)
}

// serve は QuestHandler をルータに載せてリクエストを 1 件処理する。
// 本番と同じ NewRouter を使うので、ルーティングやエラー変換（ErrorHandler）まで含めて確かめられる。
func serve(t *testing.T, h *QuestHandler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := NewRouter(NewAdventurerHandler(nil), h)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// decode はレスポンスの JSON を型 T に読み込む（ジェネリクスで、どの型にも使える）。
// t.Helper() を呼ぶと、失敗時に表示される行番号がこの関数の中ではなく呼び出し元になる。
func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	return v
}

func TestCreateQuestHandler(t *testing.T) {
	creator := &fakeCreator{}
	h := NewQuestHandler(QuestHandlerDeps{Create: creator, Clock: fixedClock{handlerNow}})

	rec := serve(t, h, http.MethodPost, "/api/quests",
		`{"title":"薬草採取","description":"森で","reward":100,"deadline":"2026-10-12T03:00:00Z"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := decode[questResponse](t, rec)
	if got.Title != "薬草採取" || got.Reward != 100 || got.Status != "open" || got.IsExpired || got.Assignee != nil {
		t.Errorf("unexpected response: %+v", got)
	}
	if got.Deadline == nil || !got.Deadline.Equal(time.Date(2026, 10, 12, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("deadline = %v", got.Deadline)
	}
	if creator.got.Deadline == nil || creator.got.Description != "森で" {
		t.Errorf("usecase input = %+v", creator.got)
	}
}

func TestCreateQuestHandler_Errors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		ucErr      error
		wantStatus int
		wantCode   string
	}{
		{"不正な JSON", `{"title":`, nil, 400, "VALIDATION_ERROR"},
		{"報酬が整数でない", `{"title":"x","reward":1.5}`, nil, 400, "VALIDATION_ERROR"},
		{"報酬なし", `{"title":"x"}`, nil, 400, "VALIDATION_ERROR"},
		{"期限の形式不正", `{"title":"x","reward":1,"deadline":"tomorrow"}`, nil, 400, "VALIDATION_ERROR"},
		{"タイトル不正", `{"title":"","reward":1}`, quest.ErrInvalidTitle, 400, "INVALID_TITLE"},
		{"報酬が負", `{"title":"x","reward":-1}`, quest.ErrInvalidReward, 400, "INVALID_REWARD"},
		{"期限が過去", `{"title":"x","reward":1,"deadline":"2000-01-01T00:00:00Z"}`, quest.ErrDeadlineInPast, 400, "DEADLINE_IN_PAST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewQuestHandler(QuestHandlerDeps{Create: &fakeCreator{err: tt.ucErr}, Clock: fixedClock{handlerNow}})
			rec := serve(t, h, http.MethodPost, "/api/quests", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := decode[errorResponse](t, rec); got.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Error.Code, tt.wantCode)
			}
		})
	}
}

type fakeLister struct {
	got usecase.ListQuestsInput
	out usecase.ListQuestsOutput
}

func (f *fakeLister) Execute(_ context.Context, in usecase.ListQuestsInput) (usecase.ListQuestsOutput, error) {
	f.got = in
	return f.out, nil
}

type fakeGetter struct {
	view usecase.QuestView
	err  error
}

func (f *fakeGetter) Execute(_ context.Context, _ uuid.UUID) (usecase.QuestView, error) {
	return f.view, f.err
}

func TestListQuestsHandler(t *testing.T) {
	past := handlerNow.Add(-time.Hour)
	expired := quest.Reconstruct(uuid.Must(uuid.NewV7()), "期限切れ", "", 1, &past, quest.StatusOpen, nil, nil, nil, past.Add(-time.Hour), 1)
	lister := &fakeLister{out: usecase.ListQuestsOutput{
		Items: []usecase.QuestView{{Quest: expired}}, Page: 2, TotalCount: 21, TotalPages: 2,
	}}
	h := NewQuestHandler(QuestHandlerDeps{List: lister, Clock: fixedClock{handlerNow}})

	rec := serve(t, h, http.MethodGet, "/api/quests?status=open&exclude_expired=true&page=2", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if lister.got.Status == nil || *lister.got.Status != quest.StatusOpen || !lister.got.ExcludeExpired || lister.got.Page != 2 {
		t.Errorf("usecase input = %+v", lister.got)
	}
	got := decode[questPageResponse](t, rec)
	if got.Page != 2 || got.PerPage != 20 || got.TotalCount != 21 || got.TotalPages != 2 || len(got.Items) != 1 {
		t.Errorf("unexpected page: %+v", got)
	}
	if !got.Items[0].IsExpired {
		t.Error("is_expired must be true for an expired open quest")
	}
}

func TestListQuestsHandler_Defaults(t *testing.T) {
	lister := &fakeLister{out: usecase.ListQuestsOutput{Items: []usecase.QuestView{}, Page: 1}}
	h := NewQuestHandler(QuestHandlerDeps{List: lister, Clock: fixedClock{handlerNow}})

	rec := serve(t, h, http.MethodGet, "/api/quests", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if lister.got.Status != nil || lister.got.ExcludeExpired || lister.got.Page != 1 {
		t.Errorf("usecase input = %+v", lister.got)
	}
	if got := decode[questPageResponse](t, rec); got.Items == nil {
		t.Error("items must be an empty array, not null")
	}
}

func TestListQuestsHandler_InvalidQuery(t *testing.T) {
	for _, q := range []string{"status=unknown", "page=0", "page=abc", "exclude_expired=maybe"} {
		t.Run(q, func(t *testing.T) {
			h := NewQuestHandler(QuestHandlerDeps{List: &fakeLister{}, Clock: fixedClock{handlerNow}})
			rec := serve(t, h, http.MethodGet, "/api/quests?"+q, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
			if got := decode[errorResponse](t, rec); got.Error.Code != "VALIDATION_ERROR" {
				t.Errorf("code = %q", got.Error.Code)
			}
		})
	}
}

func TestGetQuestHandler(t *testing.T) {
	assigneeID := uuid.Must(uuid.NewV7())
	acceptedAt := handlerNow.Add(-time.Hour)
	q := quest.Reconstruct(uuid.Must(uuid.NewV7()), "薬草採取", "森で", 100, nil, quest.StatusAccepted, &assigneeID, &acceptedAt, nil, acceptedAt, 2)
	h := NewQuestHandler(QuestHandlerDeps{
		Get:   &fakeGetter{view: usecase.QuestView{Quest: q, Assignee: adventurer.Reconstruct(assigneeID, "アリア", handlerNow)}},
		Clock: fixedClock{handlerNow},
	})

	rec := serve(t, h, http.MethodGet, "/api/quests/"+q.ID().String(), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := decode[questResponse](t, rec)
	if got.ID != q.ID().String() || got.Status != "accepted" || got.Assignee == nil || got.Assignee.Name != "アリア" ||
		got.AcceptedAt == nil || got.IsExpired {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestGetQuestHandler_Errors(t *testing.T) {
	h := NewQuestHandler(QuestHandlerDeps{Get: &fakeGetter{err: quest.ErrQuestNotFound}, Clock: fixedClock{handlerNow}})

	rec := serve(t, h, http.MethodGet, "/api/quests/not-a-uuid", "")
	if rec.Code != http.StatusBadRequest || decode[errorResponse](t, rec).Error.Code != "VALIDATION_ERROR" {
		t.Errorf("invalid uuid: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = serve(t, h, http.MethodGet, "/api/quests/"+uuid.Must(uuid.NewV7()).String(), "")
	if rec.Code != http.StatusNotFound || decode[errorResponse](t, rec).Error.Code != "QUEST_NOT_FOUND" {
		t.Errorf("not found: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

type fakeActor struct {
	gotQuestID      uuid.UUID
	gotAdventurerID uuid.UUID
	view            usecase.QuestView
	err             error
}

func (f *fakeActor) Execute(_ context.Context, questID, adventurerID uuid.UUID) (usecase.QuestView, error) {
	f.gotQuestID, f.gotAdventurerID = questID, adventurerID
	return f.view, f.err
}

func acceptedView(t *testing.T) usecase.QuestView {
	t.Helper()
	a := adventurer.Reconstruct(uuid.Must(uuid.NewV7()), "アリア", handlerNow)
	id := a.ID()
	q := quest.Reconstruct(uuid.Must(uuid.NewV7()), "薬草採取", "", 100, nil, quest.StatusAccepted, &id, &handlerNow, nil, handlerNow, 2)
	return usecase.QuestView{Quest: q, Assignee: a}
}

func TestAcceptQuestHandler(t *testing.T) {
	view := acceptedView(t)
	actor := &fakeActor{view: view}
	h := NewQuestHandler(QuestHandlerDeps{Accept: actor, Clock: fixedClock{handlerNow}})

	rec := serve(t, h, http.MethodPost, "/api/quests/"+view.Quest.ID().String()+"/accept",
		`{"adventurer_id":"`+view.Assignee.ID().String()+`"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if actor.gotQuestID != view.Quest.ID() || actor.gotAdventurerID != view.Assignee.ID() {
		t.Errorf("usecase args = %v, %v", actor.gotQuestID, actor.gotAdventurerID)
	}
	got := decode[questResponse](t, rec)
	if got.Status != "accepted" || got.Assignee == nil || got.Assignee.ID != view.Assignee.ID().String() {
		t.Errorf("unexpected response: %+v", got)
	}
}

// testActionErrors は accept / complete 共通のエラー応答を確認する。
func testActionErrors(t *testing.T, action string, newHandler func(*fakeActor) *QuestHandler, ucErrors []struct {
	err        error
	wantStatus int
	wantCode   string
}) {
	t.Helper()
	questPath := "/api/quests/" + uuid.Must(uuid.NewV7()).String() + "/" + action
	validBody := `{"adventurer_id":"` + uuid.Must(uuid.NewV7()).String() + `"}`

	for _, tt := range []struct {
		name string
		path string
		body string
	}{
		{"adventurer_id なし", questPath, `{}`},
		{"adventurer_id が UUID でない", questPath, `{"adventurer_id":"abc"}`},
		{"不正な JSON", questPath, `{`},
		{"クエスト ID が UUID でない", "/api/quests/abc/" + action, validBody},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, newHandler(&fakeActor{}), http.MethodPost, tt.path, tt.body)
			if rec.Code != http.StatusBadRequest || decode[errorResponse](t, rec).Error.Code != "VALIDATION_ERROR" {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
	for _, tt := range ucErrors {
		t.Run(tt.wantCode, func(t *testing.T) {
			rec := serve(t, newHandler(&fakeActor{err: tt.err}), http.MethodPost, questPath, validBody)
			if rec.Code != tt.wantStatus || decode[errorResponse](t, rec).Error.Code != tt.wantCode {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAcceptQuestHandler_Errors(t *testing.T) {
	testActionErrors(t, "accept",
		func(a *fakeActor) *QuestHandler {
			return NewQuestHandler(QuestHandlerDeps{Accept: a, Clock: fixedClock{handlerNow}})
		},
		[]struct {
			err        error
			wantStatus int
			wantCode   string
		}{
			{quest.ErrQuestNotOpen, 409, "QUEST_NOT_OPEN"},
			{quest.ErrQuestExpired, 409, "QUEST_EXPIRED"},
			{quest.ErrQuestNotFound, 404, "QUEST_NOT_FOUND"},
			{adventurer.ErrAdventurerNotFound, 404, "ADVENTURER_NOT_FOUND"},
		})
}

func TestCompleteQuestHandler(t *testing.T) {
	view := acceptedView(t)
	completedAt := handlerNow
	q := view.Quest
	view.Quest = quest.Reconstruct(q.ID(), q.Title(), q.Description(), q.Reward(), q.Deadline(), quest.StatusCompleted,
		q.AssigneeID(), q.AcceptedAt(), &completedAt, q.CreatedAt(), 3)
	actor := &fakeActor{view: view}
	h := NewQuestHandler(QuestHandlerDeps{Complete: actor, Clock: fixedClock{handlerNow}})

	rec := serve(t, h, http.MethodPost, "/api/quests/"+q.ID().String()+"/complete",
		`{"adventurer_id":"`+view.Assignee.ID().String()+`"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if actor.gotQuestID != q.ID() || actor.gotAdventurerID != view.Assignee.ID() {
		t.Errorf("usecase args = %v, %v", actor.gotQuestID, actor.gotAdventurerID)
	}
	got := decode[questResponse](t, rec)
	if got.Status != "completed" || got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestCompleteQuestHandler_Errors(t *testing.T) {
	testActionErrors(t, "complete",
		func(a *fakeActor) *QuestHandler {
			return NewQuestHandler(QuestHandlerDeps{Complete: a, Clock: fixedClock{handlerNow}})
		},
		[]struct {
			err        error
			wantStatus int
			wantCode   string
		}{
			{quest.ErrNotAssignee, 403, "NOT_ASSIGNEE"},
			{quest.ErrQuestNotAccepted, 409, "QUEST_NOT_ACCEPTED"},
			{quest.ErrQuestAlreadyCompleted, 409, "QUEST_ALREADY_COMPLETED"},
			{quest.ErrQuestNotFound, 404, "QUEST_NOT_FOUND"},
			{adventurer.ErrAdventurerNotFound, 404, "ADVENTURER_NOT_FOUND"},
		})
}
