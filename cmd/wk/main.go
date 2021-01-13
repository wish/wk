package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/wish/wk/pkg/kops"
	"github.com/wish/wk/pkg/opa"
)

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Set debug mode")
	rootCmd.AddCommand(clusterApplyCmd)
	clusterApplyCmd.Flags().StringP("dry", "", "", "Run dry run and save output file.")
	clusterApplyCmd.Flags().BoolP("force-update", "f", false, "Force update")
	clusterApplyCmd.Flags().BoolP("preview", "p", false, "Preview changes")
	clusterApplyCmd.Flags().BoolP("no-update", "n", false, "Create resources but don't do kops update cluster")

	opa.AddOPAOpts(clusterApplyCmd)

	rootCmd.AddCommand(clusterEditCmd)
	rootCmd.AddCommand(clusterEditIGCmd)
	rootCmd.AddCommand(channelsApplyCmd)
	channelsApplyCmd.Flags().StringP("dry", "", "", "Run dry run and save output file.")
	opa.AddOPAOpts(channelsApplyCmd)
}

var BuildSha = "BuildSha UN-SET"   // BuildSha set default value
var BuildDate = "BuildDate UN-SET" // BuildDate set default value
var rootCmd = &cobra.Command{
	Use:   "wk",
	Short: "wk is wrapper tool for managing multiple Kubernetes clusters",
	Version: "\n" +
		"  Built:\t" + BuildDate + "\n" +
		"  Git commit:\t" + BuildSha + "\n" +
		"  OS/Arch:\t" + runtime.GOOS + "/" + runtime.GOARCH,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		d, err := cmd.Flags().GetBool("debug")
		if err != nil {
			panic(err)
		}
		if d {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}

var clusterApplyCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Apply changes to cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dry, _ := cmd.Flags().GetString("dry")
		forceUpdate, _ := cmd.Flags().GetBool("force-update")
		preview, _ := cmd.Flags().GetBool("preview")
		noUpdate, _ := cmd.Flags().GetBool("no-update")
		opaQuery, err := opa.FromFlags(cmd.Flags())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := kops.ClusterApply(context.Background(), args[0], dry, forceUpdate, noUpdate, preview, opaQuery); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var clusterEditCmd = &cobra.Command{
	Use:    "cluster-edit",
	Hidden: true,
	Args:   cobra.ExactArgs(5),
	Run: func(cmd *cobra.Command, args []string) {
		if err := kops.ClusterEdit(context.Background(), args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var clusterEditIGCmd = &cobra.Command{
	Use:    "cluster-edit-ig",
	Hidden: true,
	Args:   cobra.ExactArgs(6),
	Run: func(cmd *cobra.Command, args []string) {
		if err := kops.ClusterEditIG(context.Background(), args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

var channelsApplyCmd = &cobra.Command{
	Use:   "channels",
	Short: "Apply changes to cluster's channels",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dry, _ := cmd.Flags().GetString("dry")
		opaQuery, err := opa.FromFlags(cmd.Flags())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dry = filepath.Clean(dry)
		if err := kops.ChannelsApply(context.Background(), args[0], dry, opaQuery); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
