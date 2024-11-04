package redis

import "fmt"

// Key key.
type Key string

// Format format.
func (k Key) Format(params ...interface{}) string {
	return fmt.Sprintf(string(k), params...)
}

// Common keys.

const (
	// SessionDeleteLaterMessagesForActor1 is the key for deleting later messages for actor.
	// params: actor id
	SessionDeleteLaterMessagesForActor1 Key = "session/delete_later_messages_for_actor/%d" // List
)

// CallbackQueryData keys.
const (
	// CallbackQueryData2 is the key for storing callback query data.
	// params: handler route, action hash
	CallbackQueryData2 Key = "callback_query/button_data/%s/%s"
)

// Rate limits.

const (
	// CommandRateLimitLock2 is the key for command rate limit lock.
	// params: command, platform, chat id
	CommandRateLimitLock2 Key = "rate_limit/manual_recap/command:%s/group/%s/%s"
)
