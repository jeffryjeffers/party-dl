package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() error {
	rootCmd := &cobra.Command{
		Version: "v0.0.1",
		Use:     "party-dl",
		Long:    "party-dl is a tool for downloading content from the .party sites",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}
			viper.AutomaticEnv()
			viper.SetEnvPrefix("party-dl")

			return nil
		},
	}

	rootCmd.AddCommand(downloadCmd())
	rootCmd.AddCommand(stashCmd())

	return rootCmd.ExecuteContext(context.Background())
}
