package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"time"
)

type Config struct {
	ServerAddress string `env:"RUN_ADDRESS"`
	DataBase      string `env:"DATABASE_URI"`
	Accrual       string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

const (
	defaultDataBase    = "postgres://go:1@localhost:5432/go_diplom1?sslmode=disable"
	defaultServAddr    = ":8080"
	defaultBaseUrl     = "http://localhost"
	defaultBasePort    = ":8080"
	defaultAccrualPort = ":99"
	TokenExp           = time.Hour * 3
	// TODO: вынести в env
	Secretkey = "secret"
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
		ProcessConfig.Accrual = defaultBaseUrl + defaultAccrualPort
	} else if *accrualArg != "" {
		ProcessConfig.Accrual = defaultBaseUrl + *accrualArg
	}

	return nil
}
