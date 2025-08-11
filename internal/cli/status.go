package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository and configuration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("status scaffolding executed. Implementation pending.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
