package cmd

import "github.com/spf13/cobra"

type Commander interface {
	Command() *cobra.Command // command instance
}
