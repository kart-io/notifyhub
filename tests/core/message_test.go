package core_test

import (
	"testing"

	"github.com/kart-io/notifyhub/core"
	"github.com/kart-io/notifyhub/core/message"
	"github.com/kart-io/notifyhub/tests/utils"
)

func TestMessage_NewMessage(t *testing.T) {
	msg := message.NewMessage()

	utils.AssertTrue(t, msg != nil, "message should not be nil")
	utils.AssertTrue(t, !msg.CreatedAt.IsZero(), "message should have creation time")
	// Note: ID is generated when message is built via builder pattern
}

func TestMessage_SettersAndGetters(t *testing.T) {
	msg := message.NewMessage()

	// Test Title
	msg.SetTitle("Test Title")
	utils.AssertEqual(t, "Test Title", msg.GetTitle())

	// Test Body
	msg.SetBody("Test Body")
	utils.AssertEqual(t, "Test Body", msg.GetBody())

	// Test Priority
	msg.SetPriority(message.Priority(5))
	utils.AssertEqual(t, message.Priority(5), msg.GetPriority())
}

func TestTarget_Creation(t *testing.T) {
	target := core.NewTarget(core.TargetTypeEmail, "test@example.com", "email")

	utils.AssertEqual(t, core.TargetTypeEmail, target.Type)
	utils.AssertEqual(t, "test@example.com", target.Value)
	utils.AssertEqual(t, "email", target.Platform)
}

func TestSendingResult_Creation(t *testing.T) {
	target := core.NewTarget(core.TargetTypeEmail, "test@example.com", "email")
	result := core.NewResult("msg-123", target)

	utils.AssertTrue(t, result != nil, "result should not be nil")
	utils.AssertEqual(t, "msg-123", result.MessageID)
	utils.AssertEqual(t, target.Value, result.Target.Value)
	utils.AssertEqual(t, core.StatusPending, result.Status)
}
