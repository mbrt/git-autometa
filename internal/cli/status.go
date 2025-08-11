package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show repository and configuration status",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStatus(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("status scaffolding executed. Implementation pending.")
	return nil
}
