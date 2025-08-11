package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	appconfig "git-autometa/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage git-autometa configuration",
}

var configGlobalCmd = &cobra.Command{
	Use:   "global",
	Short: "Edit or show global configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := appconfig.LoadEffectiveConfig(cfgPath)
		if err != nil {
			return err
		}
		fmt.Println("Global config handling is not implemented yet.")
		return nil
	},
}

var configRepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Edit or show repository-specific configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Repo config handling is not implemented yet.")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := appconfig.LoadEffectiveConfig(cfgPath)
		if err != nil {
			return err
		}
		// Print a minimal summary to confirm wiring
		fmt.Println("Config loaded.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGlobalCmd)
	configCmd.AddCommand(configRepoCmd)
	configCmd.AddCommand(configShowCmd)
}
