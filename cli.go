package main

import (
	"log"

	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cobra"
	"stackmachine.com/logtrace"
)

func AddCommand(root *cobra.Command, name, desc string, run func() error) {
	var command = &cobra.Command{
		Use:   name,
		Short: desc,
		Run: func(cmd *cobra.Command, args []string) {
			err := run()
			if err != nil {
				log.Fatalf("command-error %s", err)
			}
		},
	}
	root.AddCommand(command)
}

func main() {
	opentracing.InitGlobalTracer(&logtrace.Tracer{})

	var rootCmd = &cobra.Command{Use: "brewapi"}
	AddCommand(rootCmd, "migrate", "Migrate the database to the latest scheme", MigrateDatabase)
	AddCommand(rootCmd, "serve", "Start and serve the REST API", ServeWebsite)
	AddCommand(rootCmd, "sync", "Add new cards to the card database", SyncCards)
	AddCommand(rootCmd, "price", "Sync price data to the database", SyncPrices)
	AddCommand(rootCmd, "validate", "Validate price data", ValidatePrices)
	rootCmd.Execute()
}
