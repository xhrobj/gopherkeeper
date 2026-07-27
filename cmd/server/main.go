// Package main запускает HTTPS/gRPC-серверы GophKeeper.
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
	serverapp "github.com/xhrobj/gopherkeeper/internal/server/app"
	"github.com/xhrobj/gopherkeeper/internal/server/auth"
	"github.com/xhrobj/gopherkeeper/internal/server/config"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	grpcserver "github.com/xhrobj/gopherkeeper/internal/server/transport/grpc"
	httpserver "github.com/xhrobj/gopherkeeper/internal/server/transport/http"
	"github.com/xhrobj/gopherkeeper/internal/server/transport/http/middleware"
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

	httpServer := httpserver.NewServer(cfg.HTTPAddress, handler)
	grpcServer, err := grpcserver.NewServer(
		cfg.TLSCertFile,
		cfg.TLSKeyFile,
		grpcserver.Dependencies{
			Database:          pool,
			Registerer:        registrationService,
			Authenticator:     authenticationService,
			TokenValidator:    tokenManager,
			CurrentUserReader: userRepository,
			Records:           recordService,
		},
		lg,
	)
	if err != nil {
		return err
	}

	return serverapp.ServeTransports(
		ctx,
		serverapp.Transport{
			Name: "HTTPS",
			Serve: func(ctx context.Context) error {
				lg.Info("https server starting", zap.String("server_address", cfg.HTTPAddress))
				err := httpserver.ServeTLS(
					ctx,
					httpServer,
					cfg.TLSCertFile,
					cfg.TLSKeyFile,
				)
				lg.Info("https server stopped")

				return err
			},
		},
		serverapp.Transport{
			Name: "gRPC",
			Serve: func(ctx context.Context) error {
				lg.Info("grpc server starting", zap.String("server_address", cfg.GRPCAddress))
				err := grpcserver.Serve(ctx, cfg.GRPCAddress, grpcServer)
				lg.Info("grpc server stopped")

				return err
			},
		},
	)
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
