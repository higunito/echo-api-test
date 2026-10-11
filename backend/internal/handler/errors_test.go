package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid title", quest.ErrInvalidTitle, 400, "INVALID_TITLE"},
		{"invalid description", quest.ErrInvalidDescription, 400, "INVALID_DESCRIPTION"},
		{"invalid reward", quest.ErrInvalidReward, 400, "INVALID_REWARD"},
		{"deadline in past", quest.ErrDeadlineInPast, 400, "DEADLINE_IN_PAST"},
		{"quest not found", quest.ErrQuestNotFound, 404, "QUEST_NOT_FOUND"},
		{"adventurer not found", adventurer.ErrAdventurerNotFound, 404, "ADVENTURER_NOT_FOUND"},
		{"quest not open", quest.ErrQuestNotOpen, 409, "QUEST_NOT_OPEN"},
		{"quest expired", quest.ErrQuestExpired, 409, "QUEST_EXPIRED"},
		{"quest not accepted", quest.ErrQuestNotAccepted, 409, "QUEST_NOT_ACCEPTED"},
		{"quest already completed", quest.ErrQuestAlreadyCompleted, 409, "QUEST_ALREADY_COMPLETED"},
		{"not assignee", quest.ErrNotAssignee, 403, "NOT_ASSIGNEE"},
		{"wrapped domain error", fmt.Errorf("accept: %w", quest.ErrQuestNotOpen), 409, "QUEST_NOT_OPEN"},
		{"validation error", newValidationError("page は 1 以上の整数で指定してください"), 400, "VALIDATION_ERROR"},
		{"echo bind error", echo.NewHTTPError(http.StatusBadRequest, "bad json"), 400, "VALIDATION_ERROR"},
		{"echo route not found", echo.ErrNotFound, 404, "NOT_FOUND"},
		{"unknown error", errors.New("boom"), 500, "INTERNAL_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

			ErrorHandler(tt.err, c)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v (%s)", err, rec.Body.String())
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if body.Error.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

func TestErrorHandler_DoesNotLeakInternalError(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	ErrorHandler(errors.New("secret connection string"), c)

	var body errorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Message == "secret connection string" {
		t.Error("internal error message leaked to response")
	}
}
