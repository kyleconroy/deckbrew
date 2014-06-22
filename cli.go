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
			db, err := getDatabase()
			if err != nil {
				log.Fatal(err)
			}
			err = Migrate(db)
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
			db, err := getDatabase()
			if err != nil {
				log.Fatal(err)
			}
			prices, err := LoadPriceList("prices.json")
			if err != nil {
				log.Fatal(err)
			}
			m := NewApi()
			m.Map(db)
			m.Map(&prices)
			m.Run()
		},
	}
	var syncCmd = &cobra.Command{
		Use:   "load",
		Short: "Add new cards to the card database",
		Run: func(cmd *cobra.Command, args []string) {
			err := SyncDatabase(args[0])
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Loaded all data into the database")
			return
		},
	}
	var priceCmd = &cobra.Command{
		Use:   "price",
		Short: "Update card prices",
		Run: func(cmd *cobra.Command, args []string) {
			db, err := getDatabase()
			if err != nil {
				log.Fatal(err)
			}
			prices, err := LoadPriceList("prices.json")
			if err != nil {
				log.Fatal(err)
			}
			UpdatePrices(db, &prices)
		},
	}

	var rootCmd = &cobra.Command{Use: "brewapi"}
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(priceCmd)
	rootCmd.Execute()
}
