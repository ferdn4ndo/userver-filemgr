package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/ferdn4ndo/userver-filemgr/cmd/commands"
)

var rootCmd = &cobra.Command{
	Use:              "userver-filemgr",
	Short:            "uServer FileMgr API",
	TraverseChildren: true,
}

type App struct {
	*cobra.Command
}

func NewApp() App {
	cmd := App{Command: rootCmd}
	cmd.AddCommand(commands.Commands(CommonModules)...)
	cmd.AddCommand(commands.HealthProbeCommand())
	cmd.AddCommand(commands.UserverAuthCommand())
	return cmd
}

var RootApp = NewApp()

func main() {
	_ = godotenv.Load()
	if err := RootApp.Execute(); err != nil {
		os.Exit(1)
	}
}
