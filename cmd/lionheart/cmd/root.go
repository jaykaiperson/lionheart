package cmd

import (
	"github.com/spf13/cobra"

	"github.com/lionheart-vpn/lionheart/cmd/lionheart/cli"
)

var rootCmd = &cobra.Command{
	Use:   "lionheart",
	Short: "Lionheart - private, decentralized, self-hosted tunnel with a high-performance Go core and a native Android client.",
	Long:  "Run 'lionheart server' or 'lionheart client' to get started.",
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)
}

func Execute() error {
	cli.PrintBanner()
	return rootCmd.Execute()
}
