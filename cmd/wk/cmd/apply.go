package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/wish/wk/pkg/kops"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Prepare a cluster to be updated",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := kops.Apply(context.Background(), args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := kops.Update(context.Background(), args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := kops.Delete(context.Background(), args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}
