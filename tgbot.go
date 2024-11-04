package tgo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/redis/rueidis"
	"github.com/samber/lo"
	"github.com/sourcegraph/conc/panics"
	"go.uber.org/zap"

	"github.com/nekomeowww/fo"
	"github.com/nekomeowww/tgo/pkg/i18n"
	"github.com/nekomeowww/tgo/pkg/redis"
	"github.com/nekomeowww/tgo/pkg/storage/queue"
	"github.com/nekomeowww/tgo/pkg/storage/ttlcache"
	"github.com/nekomeowww/xo"
	"github.com/nekomeowww/xo/exp/channelx"
	"github.com/nekomeowww/xo/logger"
)

type botOptions struct {
	webhookURL  string
	webhookPort string
	token       string
	apiEndpoint string
	dispatcher  *Dispatcher
	logger      *logger.Logger
	queue       queue.Queue
	ttlcache    ttlcache.TTLCache
	i18n        *i18n.I18n
}

type CallOption func(*botOptions)

func WithWebhookURL(url string) CallOption {
	return func(o *botOptions) {
		o.webhookURL = url
	}
}

func WithWebhookPort(port string) CallOption {
	return func(o *botOptions) {
		o.webhookPort = port
	}
}

func WithToken(token string) CallOption {
	return func(o *botOptions) {
		o.token = token
	}
}

func WithAPIEndpoint(endpoint string) CallOption {
	return func(o *botOptions) {
		o.apiEndpoint = endpoint
	}
}

func WithDispatcher(dispatcher *Dispatcher) CallOption {
	return func(o *botOptions) {
		o.dispatcher = dispatcher
	}
}

func WithLogger(logger *logger.Logger) CallOption {
	return func(o *botOptions) {
		o.logger = logger
	}
}

func WithQueue(queue queue.Queue) CallOption {
	return func(o *botOptions) {
		o.queue = queue
	}
}

func WithTTLCache(ttlcache ttlcache.TTLCache) CallOption {
	return func(o *botOptions) {
		o.ttlcache = ttlcache
	}
}

func WithRueidis(rueidis rueidis.Client) CallOption {
	return func(o *botOptions) {
		o.queue = queue.NewRueidisQueue(rueidis)
		o.ttlcache = ttlcache.NewRueidisTTLCache(rueidis)
	}
}

func WithI18n(i18n *i18n.I18n) CallOption {
	return func(o *botOptions) {
		o.i18n = i18n
	}
}

type Bot struct {
	*tgbotapi.BotAPI

	opts       *botOptions
	logger     *logger.Logger
	dispatcher *Dispatcher
	i18n       *i18n.I18n

	webhookServer     *http.Server
	webhookUpdateChan chan tgbotapi.Update
	updateChan        tgbotapi.UpdatesChannel
	webhookStarted    bool

	alreadyStopped bool

	puller *channelx.Puller[tgbotapi.Update]
}

func NewBot(callOpts ...CallOption) (*Bot, error) {
	opts := &botOptions{
		queue:    queue.NewInMemoryQueue(),
		ttlcache: ttlcache.NewInMemoryTTLCache(),
	}

	for _, callOpt := range callOpts {
		callOpt(opts)
	}

	if opts.token == "" {
		return nil, errors.New("must supply a valid telegram bot token in configs or environment variable")
	}

	var err error
	var b *tgbotapi.BotAPI

	if opts.apiEndpoint != "" {
		b, err = tgbotapi.NewBotAPIWithAPIEndpoint(opts.token, opts.apiEndpoint+"/bot%s/%s")
	} else {
		b, err = tgbotapi.NewBotAPI(opts.token)
	}
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		BotAPI:     b,
		opts:       opts,
		logger:     opts.logger,
		dispatcher: opts.dispatcher,
	}

	bot.puller = channelx.NewPuller[tgbotapi.Update]().
		WithHandler(func(update tgbotapi.Update) {
			bot.dispatcher.Dispatch(bot.BotAPI, bot.Bot(), bot.i18n, update)
		}).
		WithPanicHandler(func(panicValues *panics.Recovered) {
			bot.logger.Error("panic occurred", zap.Any("panic", panicValues))
		})

	// init webhook server and set webhook
	if bot.opts.webhookURL != "" {
		parsed, err := url.Parse(bot.opts.webhookURL)
		if err != nil {
			return nil, err
		}

		bot.webhookUpdateChan = make(chan tgbotapi.Update, b.Buffer)
		bot.webhookServer = newWebhookServer(parsed.Path, bot.opts.webhookPort, bot.BotAPI, bot.webhookUpdateChan)
		bot.puller = bot.puller.WithNotifyChannel(bot.webhookUpdateChan)

		err = setWebhook(bot.opts.webhookURL, bot.BotAPI)
		if err != nil {
			return nil, err
		}
	} else {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		bot.updateChan = b.GetUpdatesChan(u)
		bot.puller = bot.puller.WithNotifyChannel(bot.updateChan)
	}

	// obtain webhook info
	webhookInfo, err := bot.GetWebhookInfo()
	if err != nil {
		return nil, err
	}
	if bot.opts.webhookURL != "" && webhookInfo.IsSet() && webhookInfo.LastErrorDate != 0 {
		bot.logger.Error("webhook callback failed", zap.String("last_message", webhookInfo.LastErrorMessage))
	}

	// cancel the previous set webhook
	if bot.opts.webhookURL == "" && webhookInfo.IsSet() {
		_, err := bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
		if err != nil {
			return nil, err
		}
	}

	return bot, nil
}

func (b *Bot) Stop(ctx context.Context) error {
	if b.alreadyStopped {
		return nil
	}

	b.alreadyStopped = true

	if b.opts.webhookURL != "" {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := b.webhookServer.Shutdown(closeCtx); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("failed to shutdown webhook server: %w", err)
		}

		close(b.webhookUpdateChan)
	} else {
		b.StopReceivingUpdates()
	}

	_ = b.puller.StopPull(ctx)

	return nil
}

func (b *Bot) startPullUpdates() {
	b.puller.StartPull(context.Background())
}

func (b *Bot) Start(ctx context.Context) error {
	return fo.Invoke0(ctx, func() error {
		if b.opts.webhookURL != "" && b.webhookServer != nil {
			l, err := net.Listen("tcp", b.webhookServer.Addr)
			if err != nil {
				return err
			}

			go func() {
				err := b.webhookServer.Serve(l)
				if err != nil && err != http.ErrServerClosed {
					b.logger.Fatal("", zap.Error(err))
				}
			}()

			b.logger.Info("Telegram Bot webhook server is listening", zap.String("addr", b.webhookServer.Addr))
		}

		b.startPullUpdates()
		b.webhookStarted = true

		return nil
	})
}

func (b *Bot) Bot() *BotAPI {
	return &BotAPI{
		BotAPI:   b.BotAPI,
		logger:   b.logger,
		queue:    b.opts.queue,
		ttlcache: b.opts.ttlcache,
	}
}

func (b *Bot) MayMakeRequest(endpoint string, params tgbotapi.Params) *tgbotapi.APIResponse {
	may := fo.NewMay[*tgbotapi.APIResponse]().Use(func(err error, messageArgs ...any) {
		b.logger.Error("failed to send request to telegram endpoint: "+endpoint, zap.String("request", xo.SprintJSON(params)), zap.Error(err))
	})

	return may.Invoke(b.MakeRequest(endpoint, params))
}

func (b *Bot) PinChatMessage(config PinChatMessageConfig) error {
	params, err := config.params()
	if err != nil {
		return err
	}

	b.MayMakeRequest(config.method(), params)

	return err
}

func (b *Bot) UnpinChatMessage(config UnpinChatMessageConfig) error {
	params, err := config.params()
	if err != nil {
		return err
	}

	b.MayMakeRequest(config.method(), params)

	return err
}

type BotAPI struct {
	*tgbotapi.BotAPI

	logger   *logger.Logger
	queue    queue.Queue
	ttlcache ttlcache.TTLCache
}

func (b *BotAPI) MaySend(chattable tgbotapi.Chattable) *tgbotapi.Message {
	may := fo.NewMay[tgbotapi.Message]().Use(func(err error, messageArgs ...any) {
		b.logger.Error("failed to send message to telegram", zap.String("message", xo.SprintJSON(chattable)), zap.Error(err))
	})

	return lo.ToPtr(may.Invoke(b.Send(chattable)))
}

func (b *BotAPI) MayRequest(chattable tgbotapi.Chattable) *tgbotapi.APIResponse {
	may := fo.NewMay[*tgbotapi.APIResponse]().Use(func(err error, messageArgs ...any) {
		b.logger.Error("failed to send request to telegram", zap.String("request", xo.SprintJSON(chattable)), zap.Error(err))
	})

	return may.Invoke(b.Request(chattable))
}

func (b *BotAPI) IsCannotInitiateChatWithUserErr(err error) bool {
	if err == nil {
		return false
	}

	tgbotapiErr, ok := err.(*tgbotapi.Error)
	if !ok {
		return false
	}

	return tgbotapiErr.Code == 403 && tgbotapiErr.Message == "Forbidden: bot can't initiate conversation with a user"
}

func (b *BotAPI) IsBotWasBlockedByTheUserErr(err error) bool {
	if err == nil {
		return false
	}

	tgbotapiErr, ok := err.(*tgbotapi.Error)
	if !ok {
		return false
	}

	return tgbotapiErr.Code == 403 && tgbotapiErr.Message == "Forbidden: bot was blocked by the user"
}

func (b *BotAPI) IsBotAdministrator(chatID int64) (bool, error) {
	botMember, err := b.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: chatID, UserID: b.Self.ID}})
	if err != nil {
		return false, err
	}
	if botMember.Status == string(MemberStatusAdministrator) {
		return true, err
	}

	return false, err
}

func (b *BotAPI) IsUserMemberStatus(chatID int64, userID int64, status []MemberStatus) (bool, error) {
	member, err := b.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: chatID, UserID: userID}})
	if err != nil {
		return false, err
	}
	if lo.Contains(status, MemberStatus(member.Status)) {
		return true, nil
	}

	return false, nil
}

func (b *BotAPI) IsGroupAnonymousBot(user *tgbotapi.User) bool {
	if user == nil {
		return false
	}

	return user.ID == 1087968824 && user.IsBot && user.UserName == "GroupAnonymousBot" && user.FirstName == "Group"
}

func (b *BotAPI) PushOneDeleteLaterMessage(forUserID int64, chatID int64, messageID int) error {
	if forUserID == 0 || chatID == 0 || messageID == 0 {
		return nil
	}

	err := b.queue.Push(context.Background(), redis.SessionDeleteLaterMessagesForActor1.Format(forUserID), fmt.Sprintf("%d;%d", chatID, messageID))
	if err != nil {
		b.logger.Error("failed to push one delete later message for user",
			zap.Error(err),
			zap.Int64("from_id", forUserID),
			zap.Int64("chat_id", chatID),
			zap.Int("message_id", messageID),
		)

		return err
	}

	b.logger.Debug("pushed one delete later message for user",
		zap.Int64("from_id", forUserID),
		zap.Int64("chat_id", chatID),
		zap.Int("message_id", messageID),
	)

	return nil
}

func (b *BotAPI) DeleteAllDeleteLaterMessages(forUserID int64) error {
	if forUserID == 0 {
		return nil
	}

	elems, err := b.queue.PopAll(context.Background(), redis.SessionDeleteLaterMessagesForActor1.Format(forUserID))
	if err != nil {
		return err
	}

	for _, v := range elems {
		pairs := strings.Split(v, ";")
		if len(pairs) != 2 {
			continue
		}

		chatID, err := strconv.ParseInt(pairs[0], 10, 64)
		if err != nil {
			continue
		}

		messageID, err := strconv.Atoi(pairs[1])
		if err != nil {
			continue
		}
		if chatID == 0 || messageID == 0 {
			continue
		}

		b.MayRequest(tgbotapi.NewDeleteMessage(chatID, messageID))
		b.logger.Debug("deleted one delete later message for user",
			zap.Int64("from_id", forUserID),
			zap.Int64("chat_id", chatID),
			zap.Int("message_id", messageID),
		)
	}

	return nil
}

func (b *BotAPI) AssignOneNopCallbackQueryData() (string, error) {
	return b.AssignOneCallbackQueryData("nop", "")
}

func (b *BotAPI) AssignOneCallbackQueryData(route string, data any) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	routeHash := fmt.Sprintf("%x", sha256.Sum256([]byte(route)))[0:16]
	actionHash := fmt.Sprintf("%x", sha256.Sum256(jsonData))[0:16]

	err = b.ttlcache.Set(context.Background(), redis.CallbackQueryData2.Format(route, actionHash), string(jsonData), 24*time.Hour)
	if err != nil {
		return fmt.Sprintf("%s;%s", routeHash, actionHash), err
	}

	b.logger.Debug("assigned callback query for route",
		zap.String("route", route),
		zap.String("routeHas", routeHash),
		zap.String("actionHash", actionHash),
		zap.String("data", string(jsonData)),
	)

	return fmt.Sprintf("%s;%s", routeHash, actionHash), nil
}

func (b *BotAPI) routeHashAndActionHashFromData(callbackQueryData string) (string, string) {
	handlerIdentifierPairs := strings.Split(callbackQueryData, ";")
	if len(handlerIdentifierPairs) != 2 {
		return "", ""
	}

	return handlerIdentifierPairs[0], handlerIdentifierPairs[1]
}

func (b *BotAPI) fetchCallbackQueryActionData(route string, dataHash string) (string, error) {
	str, err := b.ttlcache.Get(context.Background(), redis.CallbackQueryData2.Format(route, dataHash))
	if err != nil {
		return "", err
	}

	return str.OrEmpty(), nil
}

func (b *BotAPI) RemoveInlineKeyboardButtonFromInlineKeyboardMarkupThatMatchesDataWith(inlineKeyboardMarkup tgbotapi.InlineKeyboardMarkup, callbackData string) tgbotapi.InlineKeyboardMarkup {
	if len(inlineKeyboardMarkup.InlineKeyboard) == 0 {
		return inlineKeyboardMarkup
	}

	for i := range inlineKeyboardMarkup.InlineKeyboard {
		for j := range inlineKeyboardMarkup.InlineKeyboard[i] {
			if inlineKeyboardMarkup.InlineKeyboard[i][j].CallbackData == nil {
				continue
			}
			if *inlineKeyboardMarkup.InlineKeyboard[i][j].CallbackData == callbackData {
				inlineKeyboardMarkup.InlineKeyboard[i] = append(inlineKeyboardMarkup.InlineKeyboard[i][:j], inlineKeyboardMarkup.InlineKeyboard[i][j+1:]...)
				break
			}
		}
	}

	inlineKeyboardMarkup.InlineKeyboard = lo.Filter(inlineKeyboardMarkup.InlineKeyboard, func(row []tgbotapi.InlineKeyboardButton, _ int) bool {
		return len(row) > 0
	})

	return inlineKeyboardMarkup
}

func (b *BotAPI) ReplaceInlineKeyboardButtonFromInlineKeyboardMarkupThatMatchesDataWith(inlineKeyboardMarkup tgbotapi.InlineKeyboardMarkup, callbackData string, replacedButton tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
	if len(inlineKeyboardMarkup.InlineKeyboard) == 0 {
		return inlineKeyboardMarkup
	}

	for i := range inlineKeyboardMarkup.InlineKeyboard {
		for j := range inlineKeyboardMarkup.InlineKeyboard[i] {
			if inlineKeyboardMarkup.InlineKeyboard[i][j].CallbackData == nil {
				continue
			}
			if *inlineKeyboardMarkup.InlineKeyboard[i][j].CallbackData == callbackData {
				inlineKeyboardMarkup.InlineKeyboard[i][j] = replacedButton
				break
			}
		}
	}

	return inlineKeyboardMarkup
}
