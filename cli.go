package main

import (
	"github.com/spf13/cobra"
	"log"
)

func main() {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the database to the latest scheme",
		Run: func(cmd *cobra.Command, args []string) {
			err := MigrateDatabase()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("migrate schema to latest version")
		},
	}
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start and serve the REST API",
		Run: func(cmd *cobra.Command, args []string) {
			err := ServeWebsite()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	var syncCmd = &cobra.Command{
		Use:   "load",
		Short: "Add new cards to the card database",
		Run: func(cmd *cobra.Command, args []string) {
			err := SyncDatabase()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("load all card data into the database")
		},
	}
	var priceCmd = &cobra.Command{
		Use:   "price",
		Short: "Parse price data from TCG player",
		Run: func(cmd *cobra.Command, args []string) {
			err := GeneratePrices()
			if err != nil {
				log.Fatal(err)
			}
			log.Println("load card prices")
		},
	}
	var rootCmd = &cobra.Command{Use: "brewapi"}
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(priceCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.Execute()
}
