package connector

import (
	"fmt"
	"log"
	"os"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/utils"
)

// Config holds various command line configuration for connecting to the database
type Config struct {
	Host           *string // the host to connect to
	Socket         *string // the unix socket to connect with
	Port           *int    // the port to connect to
	User           *string // the user to connect with
	Password       *string // the password to use
	DefaultsFile   *string // name of the defaults file to use
	UseEnvironment *bool   // use the environment to set connection settings?
}

// NewConnector returns a connected Connector given the provided configuration
func NewConnector(cfg Config) *Connector {
	var defaultsFile string
	connector := new(Connector)

	if *cfg.UseEnvironment {
		connector.ConnectByEnvironment()
	} else {
		if *cfg.Host != "" || *cfg.Socket != "" {
			log.Println("--host= or --socket= defined")
			var config mysql_defaults_file.Config
			if *cfg.Host != "" && *cfg.Socket != "" {
				fmt.Println(utils.ProgName + ": Do not specify --host and --socket together")
				os.Exit(1)
			}
			if *cfg.Host != "" {
				config.Host = *cfg.Host
			}
			if *cfg.Port != 0 {
				if *cfg.Socket == "" {
					config.Port = uint16(*cfg.Port)
				} else {
					fmt.Println(utils.ProgName + ": Do not specify --socket and --port together")
					os.Exit(1)
				}
			}
			if *cfg.Socket != "" {
				config.Socket = *cfg.Socket
			}
			if *cfg.User != "" {
				config.User = *cfg.User
			}
			if *cfg.Password != "" {
				config.Password = *cfg.Password
			}
			connector.ConnectByConfig(config)
		} else {
			// no host or socket provided so assume connecting by a defaults file.
			// - if an explicit defaults-file is provided use that.
			// - if no explicit defaults-file is provided
			//   we expect to IMPLICITLY use the default
			//   defaults-file, e.g. ~/.my.cnf.
			if cfg.DefaultsFile != nil && *cfg.DefaultsFile != "" {
				log.Println("--defaults-file defined")
				defaultsFile = *cfg.DefaultsFile
			} else {
				log.Println("connecting by implicit defaults file")
			}
			connector.ConnectByDefaultsFile(defaultsFile)
		}
	}

	return connector
}
