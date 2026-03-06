package cmd

import (
	"fmt"
	"runtime"

	"github.com/abhinavcdev/gflow/internal/config"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of gflow",
	Run: func(cmd *cobra.Command, args []string) {
		ui.Banner()
		fmt.Printf("  %s %s\n", ui.BoldStyle.Render("Version:"), config.Version)
		fmt.Printf("  %s %s/%s\n", ui.BoldStyle.Render("Platform:"), runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  %s %s\n", ui.BoldStyle.Render("Go:"), runtime.Version())
		fmt.Println()
	},
}
