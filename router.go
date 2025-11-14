package tgbotapp

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler Action Enum
type HandlerAction int

const (
	CommandHandler HandlerAction = iota
	CallbackHandler
	MessageHandler
	DocumentHandler
)

func (h HandlerAction) String() string {
	switch h {
	case CommandHandler:
		return "Command Handler"
	case CallbackHandler:
		return "Callback Handler"
	case MessageHandler:
		return "Message State Handler"
	case DocumentHandler:
		return "Document Handler"
	default:
		return "Unknown Handler"
	}
}

const (
	CommandDelimiter = "|"
)

type HandlerInfo struct {
	Name string
	Type HandlerAction
	Func HandlerFunc
}

type Router interface {
	GetHandler(name string, handlerType HandlerAction) (*HandlerInfo, bool)
	AddHandler(name string, handlerType HandlerAction, f HandlerFunc) error
}

func defaultHandler(ctx *BotContext) {
	logger := ctx.Logger()

	logger.WarnContext(ctx.Ctx, "No handler found.")
}

func RouterMiddleware(router Router) Middleware {

	return RouterWithDefault(router, defaultHandler)

}

func RouterWithDefault(router Router, defaultFunc HandlerFunc) Middleware {

	return func(context *BotContext, next HandlerFunc) {
		logger := context.Logger()

		var f HandlerFunc = defaultFunc
		switch {
		case context.Update.CallbackQuery != nil:
			var action string
			callbackData := context.Update.CallbackQuery.Data
			action, context.Params = extractCallback(callbackData)

			h, ok := router.GetHandler(action, CallbackHandler)

			if !ok {
				logger.WarnContext(context.Ctx, "No handler found for callback.", "callbackName", callbackData)
				break
			}

			f = h.Func

		case context.Update.Message != nil && context.Update.Message.IsCommand():
			command := context.Update.Message.Command()
			h, ok := router.GetHandler(command, CommandHandler)

			if !ok {
				logger.WarnContext(context.Ctx, "No handler found for command.", "commandName", command)
				break
			}

			f = h.Func
			context.Params = strings.Split(context.Update.Message.CommandArguments(), CommandDelimiter)

		case context.Update.Message != nil:
			if hasDocument(context.Update.Message) {
				docType := getDocumentType(context.Update.Message)
				h, ok := router.GetHandler(docType, DocumentHandler)
				if ok {
					f = h.Func
					break
				}
			}

			if context.Session != nil {
				h, ok := router.GetHandler(string(context.Session.CurrentState()), MessageHandler)
				if !ok {
					logger.WarnContext(context.Ctx, "No handler found for message state.", "messageState", context.Session.CurrentState())
					break
				}

				f = h.Func
			}
		}

		context.SetHandler(f)
		next(context)

	}

}

func extractCallback(callbackData string) (action string, args []string) {
	if len(callbackData) < 1 {
		return
	}

	s := strings.Split(callbackData, CommandDelimiter)

	action = s[0]

	if len(s) > 1 {
		args = s[1:]
	}

	return

}

// Default Implementation for Route Table
type RouteTable struct {
	handlers map[HandlerAction]map[string]HandlerInfo
}

// AddHandler implements Router.
func (r *RouteTable) AddHandler(name string, handlerType HandlerAction, f HandlerFunc) error {
	if len(name) < 1 {
		return NewErrInvalidArgument("name must not be empty.", "name")
	}

	if _, ok := r.handlers[handlerType]; !ok {
		r.handlers[handlerType] = make(map[string]HandlerInfo)
	}

	if _, ok := r.handlers[handlerType][name]; ok {
		return NewErrHandlerAlreadyExists(name, handlerType)
	}

	r.handlers[handlerType][name] = HandlerInfo{
		Name: name,
		Type: handlerType,
		Func: f,
	}

	return nil

}

// GetHandler implements Router.
func (r *RouteTable) GetHandler(name string, handlerType HandlerAction) (*HandlerInfo, bool) {
	gp, ok := r.handlers[handlerType]
	if !ok {
		return nil, false
	}

	h, ok := gp[name]

	return &h, ok
}

func NewRouteTable() Router {
	return &RouteTable{
		handlers: make(map[HandlerAction]map[string]HandlerInfo),
	}
}

func hasDocument(message *tgbotapi.Message) bool {
	return message.Document != nil ||
		message.Photo != nil ||
		message.Video != nil ||
		message.Audio != nil ||
		message.Voice != nil ||
		message.VideoNote != nil ||
		message.Sticker != nil
}

func getDocumentType(message *tgbotapi.Message) string {
	if message.Document != nil {
		return "document"
	}
	if message.Photo != nil {
		return "photo"
	}
	if message.Video != nil {
		return "video"
	}
	if message.Audio != nil {
		return "audio"
	}
	if message.Voice != nil {
		return "voice"
	}
	if message.VideoNote != nil {
		return "video_note"
	}
	if message.Sticker != nil {
		return "sticker"
	}
	return "document"
}
