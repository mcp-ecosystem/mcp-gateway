package main

import (
	"fmt"
	"os"

	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of mock-servers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mock-servers version %s\n", version.Get())
	},
}

var rootCmd = &cobra.Command{
	Use:   "mock-servers",
	Short: "Mock Backend Servers",
	Long:  `Mock Backend Servers provide mock servers for testing`,
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Flags().StringP("transport", "t", "http", "Transport type (http or stdio or sse)")
	rootCmd.Flags().StringP("addr", "a", ":5236", "Address to listen on")
}

func run(cmd *cobra.Command, _ []string) {
	transport, _ := cmd.Flags().GetString("transport")
	addr, _ := cmd.Flags().GetString("addr")

	// We should not handle signal here, it should be handled by the server implementation
	StartMockServer(transport, addr)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
