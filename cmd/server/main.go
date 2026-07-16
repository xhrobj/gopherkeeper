// Package main запускает HTTPS-сервер GophKeeper.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/logger"
	"github.com/xhrobj/gopherkeeper/internal/server/auth"
	"github.com/xhrobj/gopherkeeper/internal/server/config"
	"github.com/xhrobj/gopherkeeper/internal/server/httpserver"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"go.uber.org/zap"
)

var (
	buildVersion = ""
	buildDate    = ""
	buildCommit  = ""
)

func main() {
	if err := printIntro(os.Stdout); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	cfg, err := config.Parse(args)
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}
	defer func() {
		_ = lg.Sync()
	}()

	pool, err := postgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	lg.Info("postgres connection verified")

	if err := migration.Run(pool); err != nil {
		return err
	}

	lg.Info("database migrations completed")

	userRepository := postgres.NewUserRepository(pool)
	recordRepository := postgres.NewRecordRepository(pool)
	passwordManager := auth.NewBcryptPasswordManager()
	tokenManager := auth.NewJWTTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	recordCrypto, err := recordcrypto.NewService(cfg.RecordMasterKey, cfg.RecordKeyID)
	if err != nil {
		return err
	}

	registrationService := service.NewRegistrationService(userRepository, passwordManager)
	authenticationService := service.NewAuthenticationService(userRepository, passwordManager, tokenManager)
	recordService := service.NewRecordService(recordRepository, recordCrypto)

	handler := middleware.WithLogging(
		httpserver.NewHandler(httpserver.Dependencies{
			Database:          pool,
			Registerer:        registrationService,
			Authenticator:     authenticationService,
			TokenValidator:    tokenManager,
			CurrentUserReader: userRepository,
			Records:           recordService,
		}),
		lg,
	)

	server := httpserver.NewServer(cfg.Address, handler)

	lg.Info(
		"https server starting",
		zap.String("server_address", cfg.Address),
	)

	if err := httpserver.ServeTLS(
		ctx,
		server,
		cfg.TLSCertFile,
		cfg.TLSKeyFile,
	); err != nil {
		return err
	}

	lg.Info("https server stopped")

	return nil
}

func printIntro(output io.Writer) error {
	if err := buildinfo.Print(output, buildinfo.Info{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	}); err != nil {
		return err
	}

	return printBanner(output)
}

func printBanner(output io.Writer) error {
	const banner = `
  ________              .__     ____  __.
 /  _____/  ____ ______ |  |__ |    |/ _|____   ____ ______   ___________
/   \  ___ /  _ \\____ \|  |  \|      <_/ __ \_/ __ \\____ \_/ __ \_  __ \
\    \_\  (  <_> )  |_> >   Y  \    |  \  ___/\  ___/|  |_> >  ___/|  | \/
 \______  /\____/|   __/|___|  /____|__ \___  >\___  >   __/ \___  >__|
        \/       |__|        \/        \/   \/     \/|__|        \/
         -= Server: Keeping your secrets secure. =-

`
	_, err := fmt.Fprint(output, banner)

	return err
}
