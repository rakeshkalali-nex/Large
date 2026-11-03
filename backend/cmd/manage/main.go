package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"buildestate/backend/internal/access/bootstrap"
	"buildestate/backend/internal/platform/config"
	"buildestate/backend/internal/platform/database"
)

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: manage migrate|bootstrap|seed:development|backup|restore:check")
	}
	switch os.Args[1] {
	case "seed:development", "backup", "restore:check":
		return fmt.Errorf("%s is unavailable until its owning feature is implemented", os.Args[1])
	case "migrate", "bootstrap":
	default:
		return fmt.Errorf("unknown operator command")
	}
	var input bootstrap.Input
	if os.Args[1] == "bootstrap" {
		var err error
		input, err = readBootstrapStdin()
		if err != nil {
			return err
		}
	}
	c, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid application configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, c.SQLitePath)
	if err != nil {
		return fmt.Errorf("database startup failed")
	}
	defer db.Close()
	if err = database.Migrate(ctx, db); err != nil {
		return fmt.Errorf("database migration failed")
	}
	if os.Args[1] == "bootstrap" {
		return runBootstrap(ctx, db, input, os.Stdout)
	}
	fmt.Println("Migrations applied successfully.")
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
