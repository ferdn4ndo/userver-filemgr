package main

import (
	"go.uber.org/fx"

	"github.com/ferdn4ndo/userver-filemgr/internal/auth"
	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/ferdn4ndo/userver-filemgr/internal/http_api"
	"github.com/ferdn4ndo/userver-filemgr/internal/media_processor"
	"github.com/ferdn4ndo/userver-filemgr/internal/media_worker"
	"github.com/ferdn4ndo/userver-filemgr/internal/object_store"
	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// CommonModules is the full application graph.
var CommonModules = fx.Options(
	lib.Module,
	fx.Provide(data.NewDB),
	fx.Provide(auth.NewService),
	fx.Provide(object_store.NewFactory),
	fx.Provide(media_processor.NewProcessor),
	fx.Provide(media_worker.NewPool),
	fx.Invoke(media_worker.Register),
	fx.Provide(http_api.NewRouter),
)
