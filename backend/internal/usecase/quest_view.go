package usecase

import (
	"github.com/higunito/echo-api-test/backend/internal/domain/adventurer"
	"github.com/higunito/echo-api-test/backend/internal/domain/quest"
)

// QuestView はクエストと受注者（未受注なら nil）の組。
//
// API のレスポンスには受注者の名前も含めたいが、Quest は受注者の ID しか持たない。
// Quest に名前を持たせると domain のモデルが表示の都合で崩れるため、usecase で 2 つを組にして返す。
type QuestView struct {
	Quest    *quest.Quest
	Assignee *adventurer.Adventurer
}
