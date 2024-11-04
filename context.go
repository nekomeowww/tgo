package tgo

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nekomeowww/tgo/pkg/i18n"
	"github.com/nekomeowww/xo/logger"
	"go.uber.org/zap"
)

type UpdateType string

const (
	UpdateTypeUnknown            UpdateType = "unknown"
	UpdateTypeMessage            UpdateType = "message"
	UpdateTypeEditedMessage      UpdateType = "edited_message"
	UpdateTypeChannelPost        UpdateType = "channel_post"
	UpdateTypeEditedChannelPost  UpdateType = "edited_channel_post"
	UpdateTypeInlineQuery        UpdateType = "inline_query"
	UpdateTypeChosenInlineResult UpdateType = "chosen_inline_result"
	UpdateTypeCallbackQuery      UpdateType = "callback_query"
	UpdateTypeShippingQuery      UpdateType = "shipping_query"
	UpdateTypePreCheckoutQuery   UpdateType = "pre_checkout_query"
	UpdateTypePoll               UpdateType = "poll"
	UpdateTypePollAnswer         UpdateType = "poll_answer"
	UpdateTypeMyChatMember       UpdateType = "my_chat_member"
	UpdateTypeChatMember         UpdateType = "chat_member"
	UpdateTypeLeftChatMember     UpdateType = "left_chat_member"
	UpdateTypeNewChatMembers     UpdateType = "new_chat_members"
	UpdateTypeChatJoinRequest    UpdateType = "chat_join_request"
	UpdateTypeChatMigrationFrom  UpdateType = "chat_migration_from"
	UpdateTypeChatMigrationTo    UpdateType = "chat_migration_to"
)

type Context struct {
	Bot    *BotAPI
	Update tgbotapi.Update
	Logger *logger.Logger
	I18n   *i18n.I18n

	mutex sync.Mutex

	abort bool

	isCallbackQuery         bool
	callBackQueryActionData string
}

func NewContext(bot *tgbotapi.BotAPI, botAPI *BotAPI, update tgbotapi.Update, logger *logger.Logger, i18n *i18n.I18n) *Context {
	return &Context{
		Bot:             botAPI,
		Update:          update,
		Logger:          logger,
		I18n:            i18n,
		isCallbackQuery: false,
	}
}

func (c *Context) UpdateType() UpdateType {
	switch {
	case c.Update.Message != nil:
		switch {
		case c.Update.Message.NewChatMembers != nil:
			return UpdateTypeNewChatMembers
		case c.Update.Message.LeftChatMember != nil:
			return UpdateTypeLeftChatMember
		case c.Update.Message.MigrateFromChatID != 0:
			return UpdateTypeChatMigrationFrom
		case c.Update.Message.MigrateToChatID != 0:
			return UpdateTypeChatMigrationTo
		default:
			return UpdateTypeMessage
		}
	case c.Update.EditedMessage != nil:
		return UpdateTypeEditedMessage
	case c.Update.ChannelPost != nil:
		return UpdateTypeChannelPost
	case c.Update.EditedChannelPost != nil:
		return UpdateTypeEditedChannelPost
	case c.Update.InlineQuery != nil:
		return UpdateTypeInlineQuery
	case c.Update.ChosenInlineResult != nil:
		return UpdateTypeChosenInlineResult
	case c.Update.CallbackQuery != nil:
		return UpdateTypeCallbackQuery
	case c.Update.ShippingQuery != nil:
		return UpdateTypeShippingQuery
	case c.Update.PreCheckoutQuery != nil:
		return UpdateTypePreCheckoutQuery
	case c.Update.Poll != nil:
		return UpdateTypePoll
	case c.Update.PollAnswer != nil:
		return UpdateTypePollAnswer
	case c.Update.MyChatMember != nil:
		return UpdateTypeMyChatMember
	case c.Update.ChatMember != nil:
		return UpdateTypeChatMember
	case c.Update.ChatJoinRequest != nil:
		return UpdateTypeChatJoinRequest
	default:
		return UpdateTypeUnknown
	}
}

func (c *Context) Abort() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.abort = true
}

func (c *Context) IsAborted() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.abort
}

func (c *Context) T(key string, args ...any) string {
	return c.I18n.TWithLanguage(c.Language(), key, args...)
}

func (c *Context) Language() string {
	if c.Update.SentFrom() == nil {
		c.Logger.Warn("update.SentFrom() is nil, fallback to 'en' language.")
		return "en"
	}

	languageCode := c.Update.SentFrom().LanguageCode
	if languageCode == "" {
		c.Logger.Warn("update.SentFrom().LanguageCode is empty, fallback to 'en' language.")
		return "en"
	}

	c.Logger.Debug("resolved language code", zap.String("languageCode", languageCode))

	return languageCode
}

func (c *Context) withCallbackQueryActionData(data string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Update.CallbackQuery == nil {
		return
	}

	c.isCallbackQuery = true
	c.callBackQueryActionData = data
}

func (c *Context) BindFromCallbackQueryData(dst any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.callBackQueryActionData == "" {
		return errors.New("empty action data")
	}

	return json.Unmarshal([]byte(c.callBackQueryActionData), dst)
}

func (c *Context) IsBotAdministrator() (bool, error) {
	return c.Bot.IsBotAdministrator(c.Update.FromChat().ID)
}

func (c *Context) IsUserMemberStatus(userID int64, status []MemberStatus) (bool, error) {
	return c.Bot.IsUserMemberStatus(c.Update.FromChat().ID, userID, status)
}

func (c *Context) RateLimitForCommand(chatID int64, command string, rate int64, perDuration time.Duration) (int64, bool, error) {
	return c.Bot.RateLimitForCommand(chatID, command, rate, perDuration)
}

func (c *Context) NewMessage(message string) MessageResponse {
	return NewMessage(c.Update.FromChat().ID, message)
}

func (c *Context) NewMessageReplyTo(message string, replyToMessageID int) MessageResponse {
	return NewMessageReplyTo(c.Update.FromChat().ID, message, replyToMessageID)
}

func (c *Context) NewEditMessageText(messageID int, text string) EditMessageResponse {
	return NewEditMessageText(c.Update.FromChat().ID, messageID, text)
}

func (c *Context) NewEditMessageTextAndReplyMarkup(messageID int, text string, replyMarkup tgbotapi.InlineKeyboardMarkup) EditMessageResponse {
	return NewEditMessageTextAndReplyMarkup(c.Update.FromChat().ID, messageID, text, replyMarkup)
}

func (c *Context) NewEditMessageReplyMarkup(messageID int, replyMarkup tgbotapi.InlineKeyboardMarkup) EditMessageResponse {
	return NewEditMessageReplyMarkup(c.Update.FromChat().ID, messageID, replyMarkup)
}
