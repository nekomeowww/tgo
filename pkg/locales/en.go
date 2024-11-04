package locales

import (
	i18nv2 "github.com/nicksnyder/go-i18n/v2/i18n"
)

func RegisterEn() []*i18nv2.Message {
	return []*i18nv2.Message{
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_route.error",
			Other: "Unable to dispatch Callback Query due to missing route DETECTED.",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_route.solution",
			Other: "For most of the time, this happens when the corresponding handler wasn't registered properly through OnCallbackQuery(...) method or internal dispatcher failed to match it, please check registered handlers and the route of them and then try again.",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_action_data.error",
			Other: "Unable to dispatch Callback Query due to missing action data DETECTED.",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_action_data.solution",
			Other: "For most of the time, this happens when the action data that stored into callback query data is either empty, not exist on cache, or failed to fetch from cache, please try to flush any corresponding cache keys and try again.",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.invalid_action_data.try_again",
			Other: "Sorry, this operation cannot be performed as it is invalid. Please initiate another session of operation and try again.",
		},
		{
			ID:    "telegram.system.commands.groups.basic.name",
			Other: "Basic Commands",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.start.help",
			Other: "Begin interacting with the bot",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.help.help",
			Other: "Display help information",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.help.message",
			Other: "Here are the available commands:\n\n{{ .Commands }}",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.cancel.help",
			Other: "Cancel any ongoing operations.",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.cancel.alreadyCancelledAll",
			Other: "No ongoing operations to cancel",
		},
	}
}
