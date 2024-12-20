package tgo

import (
	"runtime"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type errorType int

const (
	errorTypeEmptyResponse errorType = iota
	errorTypeStringResponse
)

type tgBotError interface {
	errorType() errorType
}

var (
	_ tgBotError = (*MessageError)(nil)
	_ tgBotError = (*ExceptionError)(nil)
)

type MessageError struct {
	message          string
	replyToMessageID int
	editMessage      *tgbotapi.Message
	parseMode        string
	replyMarkup      *tgbotapi.InlineKeyboardMarkup

	deleteLaterForUserID int64
	deleteLaterChatID    int64
}

func NewMessageError(message string) MessageError {
	return MessageError{
		message: message,
	}
}

func (e MessageError) errorType() errorType {
	return errorTypeStringResponse
}

func (e MessageError) Error() string {
	return e.message
}

func (e MessageError) WithReply(message *tgbotapi.Message) MessageError {
	if message == nil {
		return e
	}

	e.replyToMessageID = message.MessageID

	return e
}

func (e MessageError) WithDeleteLater(userID int64, chatID int64) MessageError {
	e.deleteLaterForUserID = userID
	e.deleteLaterChatID = chatID

	return e
}

func (e MessageError) WithEdit(message *tgbotapi.Message) MessageError {
	if message == nil {
		return e
	}

	e.editMessage = message

	return e
}

func (e MessageError) WithParseModeHTML() MessageError {
	e.parseMode = tgbotapi.ModeHTML
	return e
}

func (e MessageError) WithReplyMarkup(replyMarkup tgbotapi.InlineKeyboardMarkup) MessageError {
	e.replyMarkup = &replyMarkup
	return e
}

type ExceptionError struct {
	err              error
	message          string
	replyToMessageID int
	editMessage      *tgbotapi.Message
	callFrameSkip    int
	callFrame        *runtime.Frame
	replyMarkup      *tgbotapi.InlineKeyboardMarkup

	deleteLaterForUserID int64
	deleteLaterChatID    int64
}

func NewExceptionError(err error) ExceptionError {
	e := ExceptionError{
		err:           err,
		callFrameSkip: 1,
	}

	pc, file, line, _ := runtime.Caller(e.callFrameSkip)
	funcDetails := runtime.FuncForPC(pc)

	var funcName string
	if funcDetails != nil {
		funcName = funcDetails.Name()
	}

	e.callFrame = &runtime.Frame{
		File:     file,
		Line:     line,
		Function: funcName,
	}

	return e
}

func (e ExceptionError) errorType() errorType {
	return errorTypeEmptyResponse
}

func (e ExceptionError) Error() string {
	return e.err.Error()
}

func (e ExceptionError) WithMessage(message string) ExceptionError {
	e.message = message
	return e
}

func (e ExceptionError) WithReply(message *tgbotapi.Message) ExceptionError {
	if message == nil {
		return e
	}

	e.replyToMessageID = message.MessageID

	return e
}

func (e ExceptionError) WithEdit(message *tgbotapi.Message) ExceptionError {
	if message == nil {
		return e
	}

	e.editMessage = message

	return e
}

func (e ExceptionError) WithReplyMarkup(replyMarkup tgbotapi.InlineKeyboardMarkup) ExceptionError {
	e.replyMarkup = &replyMarkup
	return e
}

func (e ExceptionError) WithDeleteLater(userID int64, chatID int64) ExceptionError {
	e.deleteLaterForUserID = userID
	e.deleteLaterChatID = chatID

	return e
}
