package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// BuildSha set default value
var BuildSha = "BuildSha UN-SET"

// BuildDate set default value
var BuildDate = "BuildDate UN-SET"

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

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Set debug mode")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
