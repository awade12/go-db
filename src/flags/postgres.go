package flags

import (
	"flag"
	"strings"

	"go-dbs/src/databases/postgres"
)

// PostgresFlags holds all flag sets for PostgreSQL operations
type PostgresFlags struct {
	CustomFlags *flag.FlagSet
	RemoveFlags *flag.FlagSet
	Version     *string
	Port        *string
	Password    *string
	User        *string
	DBName      *string
	Volume      *string
	Memory      *string
	CPU         *string
	Name        *string
	Timezone    *string
	Locale      *string
	Networks    *string
	InitScripts *string
	SSLMode     *string
	SSLCert     *string
	SSLKey      *string
	SSLRootCert *string
	ForceRemove *bool
}

// NewPostgresFlags initializes all PostgreSQL-related flags
func NewPostgresFlags() *PostgresFlags {
	f := &PostgresFlags{
		CustomFlags: flag.NewFlagSet("create-custom", flag.ExitOnError),
		RemoveFlags: flag.NewFlagSet("remove", flag.ExitOnError),
	}

	// Initialize create-custom flags
	f.Version = f.CustomFlags.String("version", "15", "PostgreSQL version")
	f.Port = f.CustomFlags.String("port", "5432", "Port to expose")
	f.Password = f.CustomFlags.String("password", "postgres", "Database password")
	f.User = f.CustomFlags.String("user", "postgres", "Database user")
	f.DBName = f.CustomFlags.String("db", "postgres", "Database name")
	f.Volume = f.CustomFlags.String("volume", "", "Data volume path")
	f.Memory = f.CustomFlags.String("memory", "", "Memory limit")
	f.CPU = f.CustomFlags.String("cpu", "", "CPU limit")
	f.Name = f.CustomFlags.String("name", "go-dbs-postgres", "Container name")
	f.Timezone = f.CustomFlags.String("timezone", "UTC", "Container timezone")
	f.Locale = f.CustomFlags.String("locale", "en_US.utf8", "Database locale")
	f.Networks = f.CustomFlags.String("network", "", "Docker networks to join (comma-separated)")
	f.InitScripts = f.CustomFlags.String("init-script", "", "SQL scripts to run on initialization (comma-separated)")
	f.SSLMode = f.CustomFlags.String("ssl-mode", "disable", "SSL mode")
	f.SSLCert = f.CustomFlags.String("ssl-cert", "", "SSL certificate path")
	f.SSLKey = f.CustomFlags.String("ssl-key", "", "SSL private key path")
	f.SSLRootCert = f.CustomFlags.String("ssl-root-cert", "", "SSL root certificate path")

	// Initialize remove flags
	f.ForceRemove = f.RemoveFlags.Bool("force", false, "Force container removal")

	return f
}

// BuildConfig creates a PostgreSQL configuration from the flags
func (f *PostgresFlags) BuildConfig() *postgres.Config {
	var networkList []string
	if *f.Networks != "" {
		networkList = strings.Split(*f.Networks, ",")
	}

	var scriptList []string
	if *f.InitScripts != "" {
		scriptList = strings.Split(*f.InitScripts, ",")
	}

	return &postgres.Config{
		Version:       *f.Version,
		Port:          *f.Port,
		Password:      *f.Password,
		ContainerName: *f.Name,
		Username:      *f.User,
		Database:      *f.DBName,
		Volume:        *f.Volume,
		Memory:        *f.Memory,
		CPU:           *f.CPU,
		Networks:      networkList,
		InitScripts:   scriptList,
		Timezone:      *f.Timezone,
		Locale:        *f.Locale,
		SSLMode:       *f.SSLMode,
		SSLCert:       *f.SSLCert,
		SSLKey:        *f.SSLKey,
		SSLRootCert:   *f.SSLRootCert,
	}
}
