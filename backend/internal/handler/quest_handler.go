package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
	"github.com/higunito/echo-api-test/backend/internal/usecase"
)

// 以下は handler が使う usecase の形（使う側で定義する小さなインターフェース。adventurer_handler.go 参照）。
// テストでは fakeCreator などの偽物を渡す（quest_handler_test.go）。

type questCreator interface {
	Execute(ctx context.Context, in usecase.CreateQuestInput) (*quest.Quest, error)
}

type questLister interface {
	Execute(ctx context.Context, in usecase.ListQuestsInput) (usecase.ListQuestsOutput, error)
}

type questGetter interface {
	Execute(ctx context.Context, id uuid.UUID) (usecase.QuestView, error)
}

// questAction は受注・完了報告のように、冒険者がクエストに対して行う操作。
// AcceptQuest と CompleteQuest は Execute の形が同じなので、1 つのインターフェースで両方を受けられる。
type questAction interface {
	Execute(ctx context.Context, questID, adventurerID uuid.UUID) (usecase.QuestView, error)
}

// QuestHandlerDeps は QuestHandler が使う usecase。テストでは必要なものだけ設定する。
// 引数が多い関数の代わりに構造体で渡すと、名前付きで指定でき、増えても呼び出し側が読みやすい。
type QuestHandlerDeps struct {
	Create   questCreator
	List     questLister
	Get      questGetter
	Accept   questAction
	Complete questAction
	Clock    usecase.Clock // is_expired の計算に使う
}

type QuestHandler struct {
	deps QuestHandlerDeps
}

func NewQuestHandler(deps QuestHandlerDeps) *QuestHandler {
	return &QuestHandler{deps: deps}
}

// register はクエスト関連のルートを登録する（router.go から呼ばれる）。
// ":id" はパスパラメータで、ハンドラ内で c.Param("id") として取り出せる。
func (h *QuestHandler) register(g *echo.Group) {
	g.POST("/quests", h.Create)
	g.GET("/quests", h.List)
	g.GET("/quests/:id", h.Get)
	// 受注と完了報告は処理の形が同じなので、action に「どの usecase を使うか」を渡して共通化する。
	// func() questAction { ... } は無名関数（クロージャ）。登録時ではなく呼ばれたときに h.deps を読む。
	g.POST("/quests/:id/accept", h.action(func() questAction { return h.deps.Accept }))
	g.POST("/quests/:id/complete", h.action(func() questAction { return h.deps.Complete }))
}

// questResponse は API で返すクエストの JSON（openapi.yaml の Quest）。
// *time.Time や *adventurerResponse のようなポインタのフィールドは、nil なら JSON で null になる。
// time.Time は JSON にすると RFC 3339 形式（"2026-10-12T03:00:00Z"）の文字列になる。
type questResponse struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Reward      int                 `json:"reward"`
	Deadline    *time.Time          `json:"deadline"`
	Status      string              `json:"status"`
	IsExpired   bool                `json:"is_expired"`
	Assignee    *adventurerResponse `json:"assignee"`
	AcceptedAt  *time.Time          `json:"accepted_at"`
	CompletedAt *time.Time          `json:"completed_at"`
	CreatedAt   time.Time           `json:"created_at"`
}

// toQuestResponse は domain の Quest をレスポンス用の構造体に詰め替える。
// domain の型に json タグを付けず、ここで変換することで、API の形と domain の形を独立して変えられる。
func toQuestResponse(q *quest.Quest, assignee *adventurer.Adventurer, now time.Time) questResponse {
	res := questResponse{
		ID:          q.ID().String(),
		Title:       q.Title(),
		Description: q.Description(),
		Reward:      q.Reward(),
		Deadline:    q.Deadline(),
		Status:      string(q.Status()), // Status 型 → string への型変換
		IsExpired:   q.IsExpired(now),   // 保存されていない派生値なので、返すたびに計算する
		AcceptedAt:  q.AcceptedAt(),
		CompletedAt: q.CompletedAt(),
		CreatedAt:   q.CreatedAt(),
	}
	if assignee != nil {
		a := toAdventurerResponse(assignee)
		res.Assignee = &a
	}
	return res
}

// createQuestRequest は POST /api/quests のリクエスト JSON。
type createQuestRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	// *int にしているのは「reward が送られてこなかった」と「0 が送られてきた」を区別するため。
	// int だとどちらもゼロ値の 0 になってしまう。
	Reward   *int       `json:"reward"`
	Deadline *time.Time `json:"deadline"`
}

// Create は POST /api/quests（US1）。
func (h *QuestHandler) Create(c echo.Context) error {
	var req createQuestRequest
	// c.Bind はリクエストボディの JSON を req に読み込む（&req でポインタを渡して書き込んでもらう）。
	// JSON の形式が壊れている・型が合わない（reward に 1.5 など）と *echo.HTTPError(400) が返る。
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Reward == nil {
		return newValidationError("報酬を入力してください")
	}
	// タイトルの長さなどの検証は domain（NewQuest）が行うので、ここではしない。
	q, err := h.deps.Create.Execute(c.Request().Context(), usecase.CreateQuestInput{
		Title:       req.Title,
		Description: req.Description,
		Reward:      *req.Reward, // nil でないことは上で確認済み
		Deadline:    req.Deadline,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toQuestResponse(q, nil, h.deps.Clock.Now()))
}

type questPageResponse struct {
	Items      []questResponse `json:"items"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalCount int             `json:"total_count"`
	TotalPages int             `json:"total_pages"`
}

// List は GET /api/quests（US2）。
func (h *QuestHandler) List(c echo.Context) error {
	in, err := parseListQuery(c)
	if err != nil {
		return err
	}
	out, err := h.deps.List.Execute(c.Request().Context(), in)
	if err != nil {
		return err
	}
	// 同じリクエスト内で is_expired の判定がぶれないよう、現在時刻は 1 回だけ取る
	now := h.deps.Clock.Now()
	items := make([]questResponse, 0, len(out.Items))
	for _, v := range out.Items {
		items = append(items, toQuestResponse(v.Quest, v.Assignee, now))
	}
	return c.JSON(http.StatusOK, questPageResponse{
		Items:      items,
		Page:       out.Page,
		PerPage:    quest.PageSize,
		TotalCount: out.TotalCount,
		TotalPages: out.TotalPages,
	})
}

// parseListQuery はクエリパラメータ（?status=open&exclude_expired=true&page=2）を usecase の入力に変換する。
// 値が不正なら validationError（→ 400 VALIDATION_ERROR）を返す。
func parseListQuery(c echo.Context) (usecase.ListQuestsInput, error) {
	in := usecase.ListQuestsInput{Page: 1}
	if s := c.QueryParam("status"); s != "" { // 指定されていないパラメータは空文字列になる
		st, err := quest.ParseStatus(s)
		if err != nil {
			return in, newValidationError("status は open / accepted / completed のいずれかを指定してください")
		}
		in.Status = &st
	}
	if s := c.QueryParam("exclude_expired"); s != "" {
		b, err := strconv.ParseBool(s) // "true" / "false" / "1" / "0" などを bool に変換する
		if err != nil {
			return in, newValidationError("exclude_expired は true または false を指定してください")
		}
		in.ExcludeExpired = b
	}
	if s := c.QueryParam("page"); s != "" {
		p, err := strconv.Atoi(s) // Atoi = ASCII to integer。文字列 → int
		if err != nil || p < 1 {
			return in, newValidationError("page は 1 以上の整数で指定してください")
		}
		in.Page = p
	}
	return in, nil
}

// Get は GET /api/quests/:id（US2）。
func (h *QuestHandler) Get(c echo.Context) error {
	id, err := parseQuestID(c)
	if err != nil {
		return err
	}
	v, err := h.deps.Get.Execute(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toQuestResponse(v.Quest, v.Assignee, h.deps.Clock.Now()))
}

// parseQuestID はパスの :id を UUID に変換する。UUID でなければ 400。
func parseQuestID(c echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, newValidationError("クエスト ID の形式が正しくありません")
	}
	return id, nil
}

// adventurerActionRequest は受注・完了報告のリクエスト JSON（{"adventurer_id": "..."}）。
// 認証がないので、操作する冒険者はボディで受け取る（research R8）。
type adventurerActionRequest struct {
	AdventurerID string `json:"adventurer_id"`
}

// action は POST /api/quests/:id/{accept,complete} の共通処理（US3, US4）。
//
// 「ハンドラ関数を返す関数」になっている。echo.HandlerFunc は func(echo.Context) error の別名で、
// 戻り値の無名関数は引数 uc を覚えたまま（クロージャ）、リクエストのたびに呼ばれる。
func (h *QuestHandler) action(uc func() questAction) echo.HandlerFunc {
	return func(c echo.Context) error {
		questID, err := parseQuestID(c)
		if err != nil {
			return err
		}
		var req adventurerActionRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		adventurerID, err := uuid.Parse(req.AdventurerID)
		if err != nil {
			return newValidationError("adventurer_id に冒険者の ID を指定してください")
		}
		v, err := uc().Execute(c.Request().Context(), questID, adventurerID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, toQuestResponse(v.Quest, v.Assignee, h.deps.Clock.Now()))
	}
}
