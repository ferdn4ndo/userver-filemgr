package commands

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/ferdn4ndo/userver-filemgr/internal/http_api"
	"github.com/ferdn4ndo/userver-filemgr/lib"
)

type ServeCommand struct{}

func (s *ServeCommand) Short() string            { return "serve application" }
func (s *ServeCommand) Setup(cmd *cobra.Command) {}

func (s *ServeCommand) Run() lib.CommandRunner {
	return func(
		env lib.Env,
		router lib.RequestHandler,
		h *http_api.Router,
		logger lib.Logger,
		lc fx.Lifecycle,
	) *http.Server {
		server := &http.Server{
			Addr:              ":" + env.ServerPort,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      120 * time.Second,
			IdleTimeout:       180 * time.Second,
			MaxHeaderBytes:    1 << 20,
			Handler:           router.Gin,
		}
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				h.Register()
				go func() {
					logger.Info("Running server on port " + env.ServerPort)
					if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						logger.Error("Server error: ", err.Error())
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return server.Shutdown(ctx)
			},
		})
		return server
	}
}

func NewServeCommand() *ServeCommand { return &ServeCommand{} }
