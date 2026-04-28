package httpdelivery

import (
	"log/slog"
	"net/http"

	"chat-service/internal/config"
	"chat-service/internal/delivery/http/handler"
	"chat-service/internal/delivery/http/middleware"
	"chat-service/internal/usecase"
	appvalidator "chat-service/pkg/validator"
	"github.com/go-chi/chi/v5"
)

type RouterDependencies struct {
	Config         *config.Config
	Logger         *slog.Logger
	Validator      *appvalidator.Validator
	AliasUseCase   *usecase.AliasUseCase
	RoomUseCase    *usecase.RoomUseCase
	MessageUseCase *usecase.MessageUseCase
}

func NewRouter(deps RouterDependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recover(deps.Logger))
	r.Use(middleware.CORS(deps.Config.CORSAllowedOrigins))
	r.Use(middleware.Logger(deps.Logger))

	healthHandler := handler.NewHealthHandler()
	aliasHandler := handler.NewAliasHandler(deps.AliasUseCase, deps.Validator)
	roomHandler := handler.NewRoomHandler(deps.RoomUseCase, deps.Validator)
	messageHandler := handler.NewMessageHandler(deps.MessageUseCase, deps.Validator)

	r.Get("/health", healthHandler.Get)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/client-alias", aliasHandler.GetOrCreate)
		r.Post("/rooms", roomHandler.Action)
		r.Get("/rooms/status", roomHandler.Status)
		r.Post("/messages", messageHandler.Create)
		r.Get("/messages", messageHandler.List)
		r.Get("/messages/stream", messageHandler.Stream)
	})

	return r
}
