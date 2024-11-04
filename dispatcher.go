package tgo

import (
	"crypto/sha256"
	"fmt"
	"runtime/debug"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gookit/color"
	"github.com/nekomeowww/tgo/pkg/i18n"
	"github.com/nekomeowww/xo/logger"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

type Dispatcher struct {
	logger *logger.Logger

	helpCommand                *helpCommandHandler
	cancelCommand              *cancelCommandHandler
	startCommandHandler        *startCommandHandler
	middlewares                []MiddlewareFunc
	commandHandlers            map[string]HandleFunc
	channelPostHandlers        []Handler
	callbackQueryHandlers      map[string]HandleFunc
	callbackQueryHandlersRoute map[string]string
	leftChatMemberHandlers     []Handler
	newChatMembersHandlers     []Handler
	myChatMemberHandlers       []Handler
	chatMigrationFromHandlers  []Handler
}

func NewDispatcher(logger *logger.Logger) *Dispatcher {
	d := &Dispatcher{
		logger:                     logger,
		helpCommand:                newHelpCommandHandler(),
		cancelCommand:              newCancelCommandHandler(),
		startCommandHandler:        newStartCommandHandler(),
		middlewares:                make([]MiddlewareFunc, 0),
		commandHandlers:            make(map[string]HandleFunc),
		channelPostHandlers:        make([]Handler, 0),
		callbackQueryHandlers:      make(map[string]HandleFunc),
		callbackQueryHandlersRoute: make(map[string]string),
		leftChatMemberHandlers:     make([]Handler, 0),
		newChatMembersHandlers:     make([]Handler, 0),
		myChatMemberHandlers:       make([]Handler, 0),
		chatMigrationFromHandlers:  make([]Handler, 0),
	}

	d.startCommandHandler.helpCommandHandler = d.helpCommand

	d.OnCommandGroup(func(c *Context) string {
		return c.T("telegram.system.commands.groups.basic.name")
	}, []Command{
		{Command: d.helpCommand.Command(), HelpMessage: d.helpCommand.CommandHelp, Handler: NewHandler(d.helpCommand.handle)},
		{Command: d.cancelCommand.Command(), HelpMessage: d.cancelCommand.CommandHelp, Handler: NewHandler(d.cancelCommand.handle)},
		{Command: d.startCommandHandler.Command(), HelpMessage: d.startCommandHandler.CommandHelp, Handler: NewHandler(d.startCommandHandler.handle)},
	})
	d.OnCallbackQuery("nop", NewHandler(func(ctx *Context) (Response, error) {
		return nil, nil
	}))

	return d
}

func (d *Dispatcher) Use(middleware MiddlewareFunc) {
	d.middlewares = append(d.middlewares, middleware)
}

func (d *Dispatcher) OnCommand(cmd string, commandHelp func(c *Context) string, h Handler) {
	d.helpCommand.defaultGroup.commands = append(d.helpCommand.defaultGroup.commands, Command{
		Command:     cmd,
		HelpMessage: commandHelp,
	})

	d.commandHandlers[cmd] = h.Handle
}

func (d *Dispatcher) OnCommandGroup(groupName func(*Context) string, group []Command) {
	d.helpCommand.commandGroups = append(d.helpCommand.commandGroups, commandGroup{name: groupName, commands: group})

	for _, c := range group {
		d.commandHandlers[c.Command] = c.Handler.Handle
	}
}

func (d *Dispatcher) OnCancelCommand(cancelHandler func(c *Context) (bool, error), handler Handler) {
	d.cancelCommand.cancellableCommands = append(d.cancelCommand.cancellableCommands, cancellableCommand{
		shouldCancelFunc: cancelHandler,
		handler:          handler,
	})
}

func (d *Dispatcher) OnStartCommand(h Handler) {
	d.startCommandHandler.startCommandHandlers = append(d.startCommandHandler.startCommandHandlers, h)
}

func (d *Dispatcher) dispatchMessage(c *Context) {
	identityStrings := make([]string, 0)
	identityStrings = append(identityStrings, FullNameFromFirstAndLastName(c.Update.Message.From.FirstName, c.Update.Message.From.LastName))

	if c.Update.Message.From.UserName != "" {
		identityStrings = append(identityStrings, "@"+c.Update.Message.From.UserName)
	}
	if c.Update.Message.Chat.Type == "private" {
		d.logger.Debug(fmt.Sprintf("[消息｜%s] %s (%s): %s",
			MapChatTypeToChineseText(ChatType(c.Update.Message.Chat.Type)),
			strings.Join(identityStrings, " "),
			color.FgYellow.Render(c.Update.Message.From.ID),
			lo.Ternary(c.Update.Message.Text == "", "<empty or contains medias>", c.Update.Message.Text)),
		)
	} else {
		d.logger.Debug(fmt.Sprintf("[消息｜%s] [%s (%s)] %s (%s): %s",
			MapChatTypeToChineseText(ChatType(c.Update.Message.Chat.Type)),
			color.FgGreen.Render(c.Update.Message.Chat.Title),
			color.FgYellow.Render(c.Update.Message.Chat.ID),
			strings.Join(identityStrings, " "),
			color.FgYellow.Render(c.Update.Message.From.ID),
			lo.Ternary(c.Update.Message.Text == "", "<empty or contains medias>", c.Update.Message.Text)),
		)
	}
	if c.Update.Message.Command() != "" {
		d.dispatchInGoroutine(func() {
			for cmd, f := range d.commandHandlers {
				if c.Update.Message.Command() == cmd {
					_, _ = f(c)
				}
			}
		})
	}
}

func (d *Dispatcher) OnChannelPost(handler Handler) {
	d.channelPostHandlers = append(d.channelPostHandlers, handler)
}

func (d *Dispatcher) dispatchChannelPost(c *Context) {
	d.logger.Debug(fmt.Sprintf("[频道消息｜%s] [%s (%s)]: %s",
		MapChatTypeToChineseText(ChatType(c.Update.ChannelPost.Chat.Type)),
		color.FgGreen.Render(c.Update.ChannelPost.Chat.Title),
		color.FgYellow.Render(c.Update.ChannelPost.Chat.ID),
		lo.Ternary(c.Update.ChannelPost.Text == "", "<empty or contains medias>", c.Update.ChannelPost.Text),
	))

	d.dispatchInGoroutine(func() {
		for _, h := range d.channelPostHandlers {
			_, _ = h.Handle(c)
		}
	})
}

func (d *Dispatcher) OnCallbackQuery(route string, h Handler) {
	routeHash := fmt.Sprintf("%x", sha256.Sum256([]byte(route)))[0:16]
	d.callbackQueryHandlersRoute[routeHash] = route
	d.callbackQueryHandlers[routeHash] = h.Handle
}

func (d *Dispatcher) dispatchCallbackQuery(c *Context) {
	var err error
	var ok bool
	var route, routeHash, actionDataHash, actionData string

	defer func() {
		identityStrings := make([]string, 0)
		identityStrings = append(identityStrings, FullNameFromFirstAndLastName(c.Update.CallbackQuery.From.FirstName, c.Update.CallbackQuery.From.LastName))

		if c.Update.CallbackQuery.From.UserName != "" {
			identityStrings = append(identityStrings, "@"+c.Update.CallbackQuery.From.UserName)
		}

		if route == "" {
			d.logger.Error(fmt.Sprintf("[回调查询｜%s] [%s (%s)] %s (%s) : %s (Raw Data) \n%s\n\n%s\n",
				MapChatTypeToChineseText(ChatType(c.Update.CallbackQuery.Message.Chat.Type)),
				color.FgGreen.Render(c.Update.CallbackQuery.Message.Chat.Title),
				color.FgYellow.Render(c.Update.CallbackQuery.Message.Chat.ID),
				strings.Join(identityStrings, " "),
				color.FgYellow.Render(c.Update.CallbackQuery.From.ID),
				c.Update.CallbackData(),
				color.FgRed.Render(c.I18n.TWithTag(language.English, "telegram.system.callback_query.error_missing_route.error")),
				color.FgRed.Render(c.I18n.TWithTag(language.English, "telegram.system.callback_query.error_missing_route.solution")),
			),
				zap.String("route", route),
				zap.String("route_hash", routeHash),
				zap.String("action_data_hash", actionDataHash),
			)
		} else if actionData == "" {
			d.logger.Error(fmt.Sprintf("[回调查询｜%s] [%s (%s)] %s (%s) : %s (Raw Data) \n%s\n\n%s\n",
				MapChatTypeToChineseText(ChatType(c.Update.CallbackQuery.Message.Chat.Type)),
				color.FgGreen.Render(c.Update.CallbackQuery.Message.Chat.Title),
				color.FgYellow.Render(c.Update.CallbackQuery.Message.Chat.ID),
				strings.Join(identityStrings, " "),
				color.FgYellow.Render(c.Update.CallbackQuery.From.ID),
				c.Update.CallbackData(),
				color.FgRed.Render(c.I18n.TWithTag(language.English, "telegram.system.callback_query.error_missing_action_data.error")),
				color.FgRed.Render(c.I18n.TWithTag(language.English, "telegram.system.callback_query.error_missing_action_data.solution")),
			),
				zap.String("route", route),
				zap.String("route_hash", routeHash),
				zap.String("action_data_hash", actionDataHash),
			)
		} else {
			d.logger.Debug(fmt.Sprintf("[回调查询｜%s] [%s (%s)] %s (%s): %s: %s",
				MapChatTypeToChineseText(ChatType(c.Update.CallbackQuery.Message.Chat.Type)),
				color.FgGreen.Render(c.Update.CallbackQuery.Message.Chat.Title),
				color.FgYellow.Render(c.Update.CallbackQuery.Message.Chat.ID),
				strings.Join(identityStrings, " "),
				color.FgYellow.Render(c.Update.CallbackQuery.From.ID),
				route, actionData,
			),
				zap.String("route", route),
				zap.String("route_hash", routeHash),
				zap.String("action_data_hash", actionDataHash),
			)
		}
	}()

	callbackQueryActionInvalidErrMessage := tgbotapi.NewEditMessageText(c.Update.CallbackQuery.Message.Chat.ID, c.Update.CallbackQuery.Message.MessageID, c.I18n.TWithTag(language.English, "telegram.system.callback_query.invalid_action_data.try_again"))

	routeHash, actionDataHash = c.Bot.routeHashAndActionHashFromData(c.Update.CallbackQuery.Data)
	if routeHash == "" || actionDataHash == "" {
		c.Bot.MayRequest(callbackQueryActionInvalidErrMessage)
		return
	}

	route, ok = d.callbackQueryHandlersRoute[routeHash]
	if !ok || route == "" {
		return
	}

	handler, ok := d.callbackQueryHandlers[routeHash]
	if !ok || handler == nil {
		c.Bot.MayRequest(callbackQueryActionInvalidErrMessage)
		return
	}

	actionData, ok, err = d.fetchActionDataForCallbackQueryHandler(c.Bot, route, routeHash, actionDataHash)
	if err != nil {
		d.logger.Error("failed to fetch the callback query action data for handler", zap.String("route", route), zap.Error(err))
		return
	}
	if !ok {
		c.Bot.MayRequest(callbackQueryActionInvalidErrMessage)
		return
	}

	c.withCallbackQueryActionData(actionData)

	d.dispatchInGoroutine(func() {
		_, _ = handler(c)
	})
}

func (d *Dispatcher) fetchActionDataForCallbackQueryHandler(botAPI *BotAPI, route, routeHash, actionDataHash string) (string, bool, error) {
	if routeHash == "" {
		return "", false, fmt.Errorf("callback query handler route hash is empty")
	}
	if actionDataHash == "" {
		return "", false, fmt.Errorf("callback query handler action data hash is empty")
	}

	str, err := botAPI.fetchCallbackQueryActionData(route, actionDataHash)
	if err != nil {
		return "", false, err
	}

	return str, str == "", nil
}

func (d *Dispatcher) OnMyChatMember(handler Handler) {
	d.myChatMemberHandlers = append(d.myChatMemberHandlers, handler)
}

func (d *Dispatcher) dispatchMyChatMember(c *Context) {
	identityStrings := make([]string, 0)
	identityStrings = append(identityStrings, FullNameFromFirstAndLastName(c.Update.MyChatMember.From.FirstName, c.Update.MyChatMember.From.LastName))

	if c.Update.MyChatMember.From.UserName != "" {
		identityStrings = append(identityStrings, "@"+c.Update.MyChatMember.From.UserName)
	}

	oldMemberStatus := MemberStatus(c.Update.MyChatMember.OldChatMember.Status)
	newMemberStatus := MemberStatus(c.Update.MyChatMember.NewChatMember.Status)

	d.logger.Debug(fmt.Sprintf("[我的成员信息更新｜%s] [%s (%s)] %s (%s): 成员状态自 %s 变更为 %s",
		MapChatTypeToChineseText(ChatType(c.Update.MyChatMember.Chat.Type)),
		color.FgGreen.Render(c.Update.MyChatMember.Chat.Title),
		color.FgYellow.Render(c.Update.MyChatMember.Chat.ID),
		strings.Join(identityStrings, " "),
		color.FgYellow.Render(c.Update.MyChatMember.From.ID),
		MapMemberStatusToChineseText(oldMemberStatus),
		MapMemberStatusToChineseText(newMemberStatus),
	))

	switch c.Update.MyChatMember.Chat.Type {
	case "channel":
		if newMemberStatus != "administrator" {
			d.logger.Debug(fmt.Sprintf("已退出频道 %s (%d)", c.Update.MyChatMember.Chat.Title, c.Update.MyChatMember.Chat.ID))
			return
		}

		_, err := c.Bot.GetChat(tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: c.Update.MyChatMember.Chat.ID,
			},
		})
		if err != nil {
			d.logger.Error(err.Error())
			return
		}

		d.logger.Debug(fmt.Sprintf("已加入频道 %s (%d)", c.Update.MyChatMember.Chat.Title, c.Update.MyChatMember.Chat.ID))
	}

	d.dispatchInGoroutine(func() {
		for _, h := range d.myChatMemberHandlers {
			_, _ = h.Handle(c)
		}
	})
}

func (d *Dispatcher) OnLeftChatMember(h Handler) {
	d.leftChatMemberHandlers = append(d.leftChatMemberHandlers, h)
}

func (d *Dispatcher) dispatchLeftChatMember(c *Context) {
	identityStrings := make([]string, 0)
	identityStrings = append(identityStrings, FullNameFromFirstAndLastName(c.Update.Message.LeftChatMember.FirstName, c.Update.Message.LeftChatMember.LastName))

	if c.Update.Message.LeftChatMember.UserName != "" {
		identityStrings = append(identityStrings, "@"+c.Update.Message.LeftChatMember.UserName)
	}

	d.logger.Debug(fmt.Sprintf("[成员信息更新｜%s] [%s (%s)] %s (%s) 离开了聊天",
		MapChatTypeToChineseText(ChatType(c.Update.Message.Chat.Type)),
		color.FgGreen.Render(c.Update.Message.Chat.Title),
		color.FgYellow.Render(c.Update.Message.Chat.ID),
		strings.Join(identityStrings, " "),
		color.FgYellow.Render(c.Update.Message.LeftChatMember.ID),
	))

	d.dispatchInGoroutine(func() {
		for _, h := range d.leftChatMemberHandlers {
			_, _ = h.Handle(c)
		}
	})
}

func (d *Dispatcher) OnNewChatMember(h Handler) {
	d.leftChatMemberHandlers = append(d.leftChatMemberHandlers, h)
}

func (d *Dispatcher) dispatchNewChatMember(c *Context) {
	identities := make([]string, len(c.Update.Message.NewChatMembers))

	for _, identity := range c.Update.Message.NewChatMembers {
		identityStrings := make([]string, 0)
		identityStrings = append(identityStrings, FullNameFromFirstAndLastName(identity.FirstName, identity.LastName))

		if identity.UserName != "" {
			identityStrings = append(identityStrings, "@"+identity.UserName)
		}

		identityStrings = append(identityStrings, fmt.Sprintf("(%s)", color.FgYellow.Render(identity.ID)))
		identities = append(identities, strings.Join(identityStrings, " "))
	}

	d.logger.Debug(fmt.Sprintf("[成员信息更新｜%s] [%s (%s)] %s 加入了聊天",
		MapChatTypeToChineseText(ChatType(c.Update.Message.Chat.Type)),
		color.FgGreen.Render(c.Update.Message.Chat.Title),
		color.FgYellow.Render(c.Update.Message.Chat.ID),
		strings.Join(identities, ", "),
	))

	d.dispatchInGoroutine(func() {
		for _, h := range d.newChatMembersHandlers {
			_, _ = h.Handle(c)
		}
	})
}

func (d *Dispatcher) OnChatMigrationFrom(h Handler) {
	d.chatMigrationFromHandlers = append(d.chatMigrationFromHandlers, h)
}

func (d *Dispatcher) dispatchChatMigrationFrom(c *Context) {
	d.logger.Debug(fmt.Sprintf("[群组迁移] 超级群组 [%s (%s)] 已迁移自群组 [%s (%s)]",
		color.FgGreen.Render(c.Update.Message.Chat.Title),
		color.FgYellow.Render(c.Update.Message.Chat.ID),
		color.FgGreen.Render(c.Update.Message.Chat.Title),
		color.FgYellow.Render(c.Update.Message.MigrateFromChatID),
	))

	d.dispatchInGoroutine(func() {
		for _, h := range d.chatMigrationFromHandlers {
			_, _ = h.Handle(c)
		}
	})
}

func (d *Dispatcher) dispatchChatMigrationTo(c *Context) {
	d.logger.Debug(fmt.Sprintf("[群组迁移] 群组 [%s (%s)] 已迁移至超级群组 [%s (%s)]",
		color.FgGreen.Render(c.Update.Message.Chat.Title),
		color.FgYellow.Render(c.Update.Message.Chat.ID),
		color.FgGreen.Render(c.Update.Message.Chat.Title),
		color.FgYellow.Render(c.Update.Message.MigrateToChatID),
	))
}

func (d *Dispatcher) Dispatch(bot *tgbotapi.BotAPI, botAPI *BotAPI, i18n *i18n.I18n, update tgbotapi.Update) {
	for _, m := range d.middlewares {
		m(NewContext(bot, botAPI, update, d.logger, i18n), func() {})
	}

	ctx := NewContext(bot, botAPI, update, d.logger, i18n)
	switch ctx.UpdateType() {
	case UpdateTypeMessage:
		d.dispatchMessage(ctx)
	case UpdateTypeEditedMessage:
		d.logger.Debug("edited message is not supported yet")
	case UpdateTypeChannelPost:
		d.dispatchChannelPost(ctx)
	case UpdateTypeEditedChannelPost:
		d.logger.Debug("edited channel post is not supported yet")
	case UpdateTypeInlineQuery:
		d.logger.Debug("inline query is not supported yet")
	case UpdateTypeChosenInlineResult:
		d.logger.Debug("chosen inline result is not supported yet")
	case UpdateTypeCallbackQuery:
		d.dispatchCallbackQuery(ctx)
	case UpdateTypeShippingQuery:
		d.logger.Debug("shipping query is not supported yet")
	case UpdateTypePreCheckoutQuery:
		d.logger.Debug("pre checkout query is not supported yet")
	case UpdateTypePoll:
		d.logger.Debug("poll is not supported yet")
	case UpdateTypePollAnswer:
		d.logger.Debug("poll answer is not supported yet")
	case UpdateTypeMyChatMember:
		d.dispatchMyChatMember(ctx)
	case UpdateTypeChatMember:
		d.logger.Debug("chat member is not supported yet")
	case UpdateTypeLeftChatMember:
		d.dispatchLeftChatMember(ctx)
	case UpdateTypeNewChatMembers:
		d.dispatchNewChatMember(ctx)
	case UpdateTypeChatJoinRequest:
		d.logger.Debug("chat join request is not supported yet")
	case UpdateTypeChatMigrationFrom:
		d.dispatchChatMigrationFrom(ctx)
	case UpdateTypeChatMigrationTo:
		d.dispatchChatMigrationTo(ctx)
	case UpdateTypeUnknown:
		d.logger.Debug("unable to dispatch update due to unknown update type")
	default:
		d.logger.Debug("unable to dispatch update due to unknown update type", zap.String("update_type", string(ctx.UpdateType())))
	}
}

func (d *Dispatcher) dispatchInGoroutine(f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				d.logger.Error("Panic recovered from command dispatcher",
					zap.Error(fmt.Errorf("panic error: %v", err)),
					zap.Stack("stack"),
				)
				fmt.Println("Panic recovered from command dispatcher: " + string(debug.Stack()))

				return
			}
		}()

		f()
	}()
}
