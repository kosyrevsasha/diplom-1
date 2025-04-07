package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"time"
)

type ctxKey struct{}

type Config struct {
	ServerAddress string `env:"RUN_ADDRESS"`
	DataBase      string `env:"DATABASE_URI"`
	Accrual       string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	Secretkey     string `env:"JWT_KEY"`
	TokenKey      ctxKey
}

const (
	defaultDataBase    = "postgres://go:1@localhost:5432/go_diplom1?sslmode=disable"
	defaultServAddr    = ":8080"
	defaultBaseURL     = "http://localhost"
	defaultBasePort    = ":8080"
	defaultAccrualPort = ":99"
	defaultSecretkey   = "secret"
	TokenExp           = time.Hour * 3
)

var ProcessConfig Config

func Init() error {
	ProcessConfig = Config{}
	err := env.Parse(&ProcessConfig)
	if err != nil {
		return err
	}

	serverAddressArg := flag.String("a", "", "server address in format host:port")
	dbArg := flag.String("d", "", "database connection")
	accrualArg := flag.String("r", "", "path to accrual system")
	secretKeyArg := flag.String("k", "", "secret key")
	flag.Parse()

	// ServerAddress
	if ProcessConfig.ServerAddress == "" && *serverAddressArg == "" {
		ProcessConfig.ServerAddress = defaultServAddr
	} else if *serverAddressArg != "" {
		ProcessConfig.ServerAddress = *serverAddressArg
	}

	// DataBase
	if ProcessConfig.DataBase == "" && *dbArg == "" {
		ProcessConfig.DataBase = defaultDataBase
	} else if *dbArg != "" {
		ProcessConfig.DataBase = *dbArg
	}

	// Accural
	if ProcessConfig.Accrual == "" && *accrualArg == "" {
		ProcessConfig.Accrual = defaultBaseURL + defaultAccrualPort
	} else if *accrualArg != "" {
		ProcessConfig.Accrual = defaultBaseURL + *accrualArg
	}

	// SecretKey
	if ProcessConfig.Secretkey == "" && *secretKeyArg == "" {
		ProcessConfig.Secretkey = defaultSecretkey
	} else if *secretKeyArg != "" {
		ProcessConfig.Secretkey = *secretKeyArg
	}

	return nil
}
