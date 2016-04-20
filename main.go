package main

import (
	"log"
	"net/http"

	"stackmachine.com/logtrace"
	"stackmachine.com/vhost"

	"github.com/kyleconroy/deckbrew/api"
	"github.com/kyleconroy/deckbrew/config"
	"github.com/kyleconroy/deckbrew/image"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cobra"
)

func addCommand(root *cobra.Command, name, desc string, run func() error) {
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
	log.SetFlags(0)
	opentracing.InitGlobalTracer(&logtrace.Tracer{})

	var rootCmd = &cobra.Command{Use: "deckbrew"}
	addCommand(rootCmd, "migrate", "Migrate the database to the latest scheme", api.MigrateDatabase)
	addCommand(rootCmd, "serve", "Start and serve the REST API", Serve)
	addCommand(rootCmd, "sync", "Add new cards to the card database", api.SyncCards)
	rootCmd.Execute()
}

func Serve() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	return http.ListenAndServe(":"+cfg.Port, vhost.Handler{
		cfg.HostAPI:   api.New(cfg),
		cfg.HostImage: image.New(),
	})
}
