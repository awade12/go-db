package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/awade12/go-db/src/databases/postgres"
	"github.com/awade12/go-db/src/flags"
)

func printUsage() {
	fmt.Println("Usage: go-db <command> <database-type> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  create         Create a new database (easy mode)")
	fmt.Println("  create-custom  Create a new database with custom configuration")
	fmt.Println("  start          Start a stopped database")
	fmt.Println("  stop           Stop a running database")
	fmt.Println("  remove         Remove a database container")
	fmt.Println("\nDatabase Types:")
	fmt.Println("  postgres       PostgreSQL database")
	fmt.Println("\nCustom Mode Options (for create-custom):")
	fmt.Println("  --version      PostgreSQL version (default: 15)")
	fmt.Println("  --port         Port to expose (default: 5432)")
	fmt.Println("  --password     Database password")
	fmt.Println("  --user         Database user")
	fmt.Println("  --db           Database name")
	fmt.Println("  --volume       Data volume path for persistence")
	fmt.Println("  --memory       Memory limit (e.g., '1g')")
	fmt.Println("  --cpu          CPU limit (e.g., '0.5')")
	fmt.Println("  --name         Container name")
	fmt.Println("  --timezone     Container timezone (default: UTC)")
	fmt.Println("  --locale       Database locale (default: en_US.utf8)")
	fmt.Println("  --network      Docker network to join (can be specified multiple times)")
	fmt.Println("  --init-script  SQL script to run on initialization (can be specified multiple times)")
	fmt.Println("  --ssl-mode     SSL mode (disable, require, verify-ca, verify-full)")
	fmt.Println("  --ssl-cert     Path to SSL certificate")
	fmt.Println("  --ssl-key      Path to SSL private key")
	fmt.Println("  --ssl-root-cert Path to SSL root certificate")
	fmt.Println("\nManagement Commands:")
	fmt.Println("  start <name>   Start a stopped database container")
	fmt.Println("  stop <name>    Stop a running database container")
	fmt.Println("  remove <name>  Remove a database container (use --force to force removal)")
	fmt.Println("\nExamples:")
	fmt.Println("  go-db create postgres")
	fmt.Println("  go-db create-custom postgres --version 15 --port 5432 --password mysecretpassword --user myuser --db mydb --volume /data/postgres")
	fmt.Println("  go-db start mydb")
	fmt.Println("  go-db stop mydb")
	fmt.Println("  go-db remove mydb --force")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])

	// Initialize flags
	postgresFlags := flags.NewPostgresFlags()

	// Handle different commands
	switch command {
	case "create":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		dbType := strings.ToLower(os.Args[2])
		switch dbType {
		case "postgres":
			if err := postgres.Create(); err != nil {
				fmt.Printf("Error creating PostgreSQL database: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Printf("Unsupported database type: %s\n", dbType)
			os.Exit(1)
		}

	case "create-custom":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		dbType := strings.ToLower(os.Args[2])
		switch dbType {
		case "postgres":
			postgresFlags.CustomFlags.Parse(os.Args[3:])
			cfg := postgresFlags.BuildConfig()
			if err := postgres.CreateWithConfig(cfg); err != nil {
				fmt.Printf("Error creating PostgreSQL database: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Printf("Unsupported database type: %s\n", dbType)
			os.Exit(1)
		}

	case "start":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		if err := postgres.Start(os.Args[2]); err != nil {
			fmt.Printf("Error starting container: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		if err := postgres.Stop(os.Args[2]); err != nil {
			fmt.Printf("Error stopping container: %v\n", err)
			os.Exit(1)
		}

	case "remove":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		postgresFlags.RemoveFlags.Parse(os.Args[3:])
		if err := postgres.Remove(os.Args[2], *postgresFlags.ForceRemove); err != nil {
			fmt.Printf("Error removing container: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}
