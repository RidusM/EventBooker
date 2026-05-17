package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ebooker/internal/config"
	"ebooker/internal/repository"
	"ebooker/internal/service"
	handler "ebooker/internal/transport/http"
	"ebooker/internal/transport/notifier"

	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/dbpg/pgx-driver/transaction"
	"github.com/wb-go/wbf/logger"
	"golang.org/x/sync/errgroup"
)

const (
	_defaultExpiryInterval = 30 * time.Second
)

func Run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	var (
		db  *pgxdriver.Postgres
		err error
	)

	defer func() {
		closeResources(ctx, db, log)
	}()

	db, err = initDatabase(&cfg.Database, log)
	if err != nil {
		return err
	}

	tm, err := transaction.NewManager(db, log)
	if err != nil {
		return fmt.Errorf("init transaction manager: %w", err)
	}

	bookingSvc, userSvc, handler, tgNotifier, err := initServices(ctx, cfg, db, tm, log)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	startWorkers(ctx, eg, bookingSvc, userSvc, handler, tgNotifier, cfg, log)

	if egErr := eg.Wait(); egErr != nil && !errors.Is(egErr, context.Canceled) {
		return fmt.Errorf("app execution failed: %w", egErr)
	}

	return nil
}

func closeResources(ctx context.Context, db *pgxdriver.Postgres, log logger.Logger) {
	if db != nil {
		db.Close()
		log.LogAttrs(ctx, logger.InfoLevel, "database connection closed")
	}
}

func initServices(
	ctx context.Context,
	cfg *config.Config,
	db *pgxdriver.Postgres,
	tm transaction.Manager,
	log logger.Logger,
) (*service.BookingService, *service.UserService, *handler.BookingHandler, *notifier.TelegramNotifier, error) {
	eventRepo := repository.NewEventRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	userRepo := repository.NewUserRepository(db)

	emailNotifier := notifier.NewEmailNotifier(
		cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From, log,
	)

	tgNotifier, err := notifier.NewTelegramNotifier(cfg.TG.Token, log)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("init telegram notifier: %w", err)
	}

	multiNotifier := notifier.NewMultiNotifier(emailNotifier, tgNotifier)
	log.LogAttrs(ctx, logger.InfoLevel, "multi-notifier initialized with email and telegram")

	eventSvc := service.NewEventService(eventRepo, tm, log, service.BookingTTLMin(cfg.Service.BookingTTLMins))

	userSvc := service.NewUserService(userRepo, tm, log)

	bookingSvc := service.NewBookingService(bookingRepo, eventRepo, userRepo, tm, multiNotifier, eventSvc, log)

	httpHandler := handler.NewBookingHandler(eventSvc, bookingSvc, userSvc, log, cfg.TG)

	return bookingSvc, userSvc, httpHandler, tgNotifier, nil
}

func startWorkers(
	ctx context.Context,
	eg *errgroup.Group,
	bookingSvc *service.BookingService,
	userSvc *service.UserService,
	httpHandler *handler.BookingHandler,
	tgNotifier *notifier.TelegramNotifier,
	cfg *config.Config,
	log logger.Logger,
) {
	eg.Go(func() error {
		return startHTTPServer(ctx, httpHandler, &cfg.HTTP, log)
	})

	interval := _defaultExpiryInterval
	eg.Go(func() error {
		log.LogAttrs(ctx, logger.InfoLevel, "starting expiry worker", logger.Duration("interval", interval))
		bookingSvc.RunExpiryWorker(ctx, interval)
		return nil
	})

	if tgNotifier != nil {
		eg.Go(func() error {
			log.LogAttrs(ctx, logger.InfoLevel, "starting telegram polling for subscribers")
			tgHandler := userSvc.GetTelegramStartHandler()
			tgNotifier.StartPolling(
				ctx,
				func(ctx context.Context, username string, chatID *int64, startPayload string) error {
					return tgHandler(ctx, username, chatID, startPayload)
				},
			)
			return nil
		})
	}
}

func startHTTPServer(ctx context.Context, h *handler.BookingHandler, cfg *config.HTTP, log logger.Logger) error {
	server := handler.NewHTTPServer(h, cfg, log)
	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	return nil
}

func initDatabase(cfg *config.Database, log logger.Logger) (*pgxdriver.Postgres, error) {
	db, err := pgxdriver.New(
		cfg.DSN,
		log,
		pgxdriver.MaxPoolSize(cfg.PoolMax),
		pgxdriver.MaxConnAttempts(cfg.ConnAttempts),
		pgxdriver.BaseRetryDelay(cfg.BaseRetryDelay),
		pgxdriver.MaxRetryDelay(cfg.MaxRetryDelay),
	)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	return db, nil
}
