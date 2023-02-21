package connector

import (
	"fmt"
	"log"
	"os"

	"github.com/sjmudd/mysql_defaults_file"
	"github.com/sjmudd/ps-top/utils"
)

// Config holds various command line flags related to connecting to the database
type Config struct {
	Host           *string // the host to connect to
	Socket         *string // the unix socket to connect with
	Port           *int    // the port to connect to
	User           *string // the user to connect with
	Password       *string // the password to use
	DefaultsFile   *string // name of the defaults file to use
	UseEnvironment *bool   // use the environment to set connection settings?
}

// NewConnector returns a connected Connector given the provided flags
func NewConnector(flags Config) *Connector {
	var defaultsFile string
	connector := new(Connector)

	if *flags.UseEnvironment {
		connector.ConnectByEnvironment()
	} else {
		if *flags.Host != "" || *flags.Socket != "" {
			log.Println("--host= or --socket= defined")
			var config mysql_defaults_file.Config
			if *flags.Host != "" && *flags.Socket != "" {
				fmt.Println(utils.ProgName + ": Do not specify --host and --socket together")
				os.Exit(1)
			}
			if *flags.Host != "" {
				config.Host = *flags.Host
			}
			if *flags.Port != 0 {
				if *flags.Socket == "" {
					config.Port = uint16(*flags.Port)
				} else {
					fmt.Println(utils.ProgName + ": Do not specify --socket and --port together")
					os.Exit(1)
				}
			}
			if *flags.Socket != "" {
				config.Socket = *flags.Socket
			}
			if *flags.User != "" {
				config.User = *flags.User
			}
			if *flags.Password != "" {
				config.Password = *flags.Password
			}
			connector.ConnectByConfig(config)
		} else {
			// no host or socket provided so assume connecting by a defaults file.
			// - if an explicit defaults-file is provided use that.
			// - if no explicit defaults-file is provided
			//   we expect to IMPLICITLY use the default
			//   defaults-file, e.g. ~/.my.cnf.
			if flags.DefaultsFile != nil && *flags.DefaultsFile != "" {
				log.Println("--defaults-file defined")
				defaultsFile = *flags.DefaultsFile
			} else {
				log.Println("connecting by implicit defaults file")
			}
			connector.ConnectByDefaultsFile(defaultsFile)
		}
	}

	return connector
}
