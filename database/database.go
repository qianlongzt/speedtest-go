package database

import (
	"fmt"

	"github.com/librespeed/speedtest/config"
	"github.com/librespeed/speedtest/database/bolt"
	"github.com/librespeed/speedtest/database/memory"
	"github.com/librespeed/speedtest/database/mysql"
	"github.com/librespeed/speedtest/database/none"
	"github.com/librespeed/speedtest/database/postgresql"
	"github.com/librespeed/speedtest/database/schema"
)

var (
	DB schema.DataAccess
)

type opener func(schema.Config) (schema.DataAccess, error)

var dbTypeMap = map[string]opener{
	"postgresql": postgresql.Open,
	"mysql":      mysql.Open,
	"bolt":       bolt.Open,
	"memory":     memory.Open,
	"none":       none.Open,
}

func SetDBInfo(conf *config.Config) error {
	open, ok := dbTypeMap[conf.DatabaseType]
	if !ok {
		panic(fmt.Errorf("unsupported database type: %s", conf.DatabaseType))
	}
	var err error
	DB, err = open(schema.Config{
		File:     conf.DatabaseFile,
		Hostname: conf.DatabaseHostname,
		Username: conf.DatabaseUsername,
		Password: conf.DatabasePassword,
		Database: conf.DatabaseName,
	})
	if err != nil {
		return err
	}
	return nil
}
