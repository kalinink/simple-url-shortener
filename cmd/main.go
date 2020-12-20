package main

import (
	"context"
	"fmt"
	"github.com/kalinink/simple-url-shortener/internal/database"
	"github.com/kalinink/simple-url-shortener/internal/handler"
	"github.com/kalinink/simple-url-shortener/internal/repository"
	"github.com/kalinink/simple-url-shortener/internal/shortener"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var serviceVersion = "dev"

type config struct {
	DBConnStr             string        `envconfig:"DB_CONN_STR" required:"true"`
	DBMaxConnections      int           `envconfig:"DB_MAX_CONNECTIONS" default:"20"`
	DBMaxIdleConns        int           `envconfig:"DB_MAX_IDLE_CONNECTIONS" default:"0"`
	DBConnMaxLifetime     time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
	DBReadTimeout         time.Duration `envconfig:"DB_READ_TIMEOUT_SEC" default:"1m"`
	DBNumberInitConnRetry int           `envconfig:"DB_NUMBER_CONNECTIONS_RETRY" default:"6"`

	ServerPort             string        `envconfig:"SERVER_PORT" default:"8080"`
	ServerHost             string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	ServerReadTimeout      time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"30s"`
	ServerWriteTimeout     time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	GracefulShutdownPeriod time.Duration `envconfig:"GRACE_PERIOD" default:"10s"`

	HostName    string        `envconfig:"HOST_NAME" default:"example.com"`
	HTTPScheme  string        `envconfig:"HTTP_SCHEME" default:"http"`
	URLLifeTime time.Duration `envconfig:"URL_LIFE_TIME" default:"24h"`
}

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	log.Info().Msgf("Version: %s", serviceVersion)

	if err := run(&log); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run(log *zerolog.Logger) error {
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		return err
	}

	dbConn, err := database.Connect(cfg.DBConnStr, database.Config{
		MaxOpenConns:           cfg.DBMaxConnections,
		ConnMaxLifetime:        cfg.DBConnMaxLifetime,
		MaxIdleConns:           cfg.DBMaxIdleConns,
		NumberInitConnectRetry: cfg.DBNumberInitConnRetry,
	})

	if err != nil {
		return fmt.Errorf("db connect err: %s", err.Error())
	}
	defer func() { _ = dbConn.Close() }()

	err = database.MakeMigrations(dbConn)
	if err != nil {
		return fmt.Errorf("make migrations: %s", err.Error())
	}

	store := repository.NewURL(dbConn, cfg.DBReadTimeout)
	service := shortener.NewService(store, cfg.HostName, cfg.HTTPScheme, cfg.URLLifeTime, log)

	apiAddr := net.JoinHostPort(cfg.ServerHost, cfg.ServerPort)
	serverHTTP := http.Server{
		Addr:         apiAddr,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
	}

	httpHandler := handler.NewHTTPHandler(service, log)

	serverErr := make(chan error, 1)
	go func() {
		log.Info().Msgf("Start listen HTTP server on %s", apiAddr)
		serverErr <- httpHandler.RegisterAndStartServer(&serverHTTP)
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-shutdown:
		log.Info().Msgf("exiting: got signal %s", sig)

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownPeriod)
		defer cancel()

		if err := serverHTTP.Shutdown(ctx); err != nil {
			return fmt.Errorf("server graceful shutdown: %w", err)
		}

		log.Info().Msg("HTTP server stop")
		return nil

	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("stopping the server with an error: %w", err)
		}

		return nil
	}
}
