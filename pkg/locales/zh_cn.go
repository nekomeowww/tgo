package locales

import (
	i18nv2 "github.com/nicksnyder/go-i18n/v2/i18n"
)

func RegisterZhCN() []*i18nv2.Message {
	return []*i18nv2.Message{
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_route.error",
			Other: "无法调度 Callback Query，检测到缺少路由。",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_route.solution",
			Other: "大多数情况下，发生这种情况的原因是相应的处理程序没有通过 OnCallbackQuery(...) 方法正确注册，或者内部派发器未能与之匹配，请检查已注册的处理程序及其路由，然后再试一次。",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_action_data.error",
			Other: "无法调度 Callback Query，检测到缺少操作数据。",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.error_missing_action_data.solution",
			Other: "大多数情况下，当存储在回调查询数据中的操作数据为空、不存在于缓存中或无法从缓存中获取时会出现这种情况，请尝试刷新相应的缓存键并重试。",
		},
		{
			ID:    "telegram.system.dispatch.callback_query.invalid_action_data.try_again",
			Other: "抱歉，因为操作无效，此操作无法进行，请重新发起操作后再试。",
		},
		{
			ID:    "telegram.system.commands.groups.basic.name",
			Other: "基础命令",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.start.help",
			Other: "开始与 Bot 的交互",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.help.help",
			Other: "获取帮助",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.help.message",
			Other: "当前支持这些命令：\n\n{{ .Commands }}",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.cancel.help",
			Other: "取消当前操作",
		},
		{
			ID:    "telegram.system.commands.groups.basic.commands.cancel.alreadyCancelledAll",
			Other: "已经没有正在进行的操作了",
		},
	}
}
