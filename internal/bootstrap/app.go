package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"chat-service/internal/config"
	httpdelivery "chat-service/internal/delivery/http"
	"chat-service/internal/infrastructure/postgres"
	redisinfra "chat-service/internal/infrastructure/redis"
	"chat-service/internal/usecase"
	appvalidator "chat-service/pkg/validator"
)

type App struct {
	Handler     http.Handler
	postgresDB  *postgres.DB
	redisClient *redisinfra.Client
}

func NewApp(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	redisClient, err := redisinfra.NewClient(ctx, cfg)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	roomRepo := postgres.NewRoomRepository(db.Gorm())
	messageRepo := postgres.NewMessageRepository(db.Gorm())
	aliasRepo := postgres.NewAliasRepository(db.Gorm())
	pubSub := redisinfra.NewPubSub(redisClient)

	aliasUseCase := usecase.NewAliasUseCase(aliasRepo)
	roomUseCase := usecase.NewRoomUseCase(roomRepo, messageRepo, cfg.RoomDefaultTTL)
	messageUseCase := usecase.NewMessageUseCase(messageRepo, roomRepo, aliasRepo, pubSub)

	handler := httpdelivery.NewRouter(httpdelivery.RouterDependencies{
		Config:         cfg,
		Logger:         log,
		Validator:      appvalidator.New(),
		AliasUseCase:   aliasUseCase,
		RoomUseCase:    roomUseCase,
		MessageUseCase: messageUseCase,
	})

	return &App{
		Handler:     handler,
		postgresDB:  db,
		redisClient: redisClient,
	}, nil
}

func (a *App) Close(ctx context.Context) error {
	var closeErr error

	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			closeErr = err
		}
	}

	if a.postgresDB != nil {
		if err := a.postgresDB.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	_ = ctx
	return closeErr
}
