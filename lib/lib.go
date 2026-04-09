package lib

import "go.uber.org/fx"

// Module exports root dependencies for the HTTP application.
var Module = fx.Options(
	fx.Provide(NewRequestHandler),
	fx.Provide(NewEnv),
	fx.Provide(GetLogger),
	fx.Provide(NewDatabase),
)
