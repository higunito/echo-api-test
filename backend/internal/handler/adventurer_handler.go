package handler

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
)

// adventurerLister は handler が必要とする usecase の形。
//
// usecase.ListAdventurers（具体的な型）ではなく小さなインターフェースで受け取るのは、
// テストで偽物に差し替えられるようにするため。Go では「インターフェースは使う側で定義する」のが慣習で、
// usecase 側は何も宣言しなくても、Execute メソッドを持っていればこのインターフェースを満たす。
type adventurerLister interface {
	Execute(ctx context.Context) ([]*adventurer.Adventurer, error)
}

type AdventurerHandler struct {
	list adventurerLister
}

func NewAdventurerHandler(list adventurerLister) *AdventurerHandler {
	return &AdventurerHandler{list: list}
}

// adventurerResponse は API で返す冒険者の JSON。クエストの受注者（assignee）にも使う。
type adventurerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toAdventurerResponse(a *adventurer.Adventurer) adventurerResponse {
	return adventurerResponse{ID: a.ID().String(), Name: a.Name()}
}

// List は GET /api/adventurers。
//
// Echo のハンドラは func(c echo.Context) error の形。
// c からリクエストを読み、c.JSON(...) でレスポンスを書く。エラーは return すれば ErrorHandler が処理する。
func (h *AdventurerHandler) List(c echo.Context) error {
	// c.Request().Context() はこのリクエスト専用の context。クライアントが切断するとキャンセルされる。
	as, err := h.list.Execute(c.Request().Context())
	if err != nil {
		return err
	}
	items := make([]adventurerResponse, 0, len(as))
	for _, a := range as {
		items = append(items, toAdventurerResponse(a))
	}
	// map[string]any は「キーが文字列、値は何でもよい」マップ。1 回しか使わない小さな JSON に便利。
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}
