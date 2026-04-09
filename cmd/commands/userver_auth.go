package commands

import (
	"github.com/spf13/cobra"

	"github.com/ferdn4ndo/userver-filemgr/internal/userver_auth"
	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// UserverAuthCommand wires bootstrap:auth (no Fx / DB).
func UserverAuthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap:auth",
		Short: "Create uServer-Auth system and optional first admin (HTTP only)",
		Long: `Calls uServer-Auth HTTP API: POST /auth/system (Token: SYSTEM_CREATION_TOKEN)
and optionally POST /auth/register. Controlled by env; see .env.template.

On success, updates ENV_FILE (default .env) only for keys that are empty or placeholder
there; existing non-empty values are left unchanged. Disabled by FILEMGR_SKIP_PERSIST_BOOTSTRAP_ENV=1.

Use SKIP_AUTH_BOOTSTRAP=1 to skip. Safe when variables are unset (no-op).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return userver_auth.Run(cmd.OutOrStdout(), lib.NewEnv())
		},
	}
}
