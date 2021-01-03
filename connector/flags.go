package connector

import (
	"fmt"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"os"
)

// Flags holds various command line flags related to connecting to the database
type Flags struct {
	Host           *string // host to connect to
	Socket         *string // unix socket to connect with
	Port           *int    // port to connect to
	User           *string // user to connect with
	Password       *string // password to use
	DefaultsFile   *string // defaults file to use
	UseEnvironment *bool   // use the environment to set connection settings
}

// NewConnector returns a connected Connector given the provided flags
func NewConnector(flags Flags) *Connector {
	var defaultsFile string
	connector := new(Connector)

	if *flags.UseEnvironment {
		connector.ConnectByEnvironment()
	} else {
		if *flags.Host != "" || *flags.Socket != "" {
			logger.Println("--host= or --socket= defined")
			var components = make(map[string]string)
			if *flags.Host != "" && *flags.Socket != "" {
				fmt.Println(lib.ProgName + ": Do not specify --host and --socket together")
				os.Exit(1)
			}
			if *flags.Host != "" {
				components["host"] = *flags.Host
			}
			if *flags.Port != 0 {
				if *flags.Socket == "" {
					components["port"] = fmt.Sprintf("%d", *flags.Port)
				} else {
					fmt.Println(lib.ProgName + ": Do not specify --socket and --port together")
					os.Exit(1)
				}
			}
			if *flags.Socket != "" {
				components["socket"] = *flags.Socket
			}
			if *flags.User != "" {
				components["user"] = *flags.User
			}
			if *flags.Password != "" {
				components["password"] = *flags.Password
			}
			connector.ConnectByComponents(components)
		} else {
			if flags.DefaultsFile != nil && *flags.DefaultsFile != "" {
				logger.Println("--defaults-file defined")
				defaultsFile = *flags.DefaultsFile
			} else {
				logger.Println("connecting by implicit defaults file")
			}
			connector.ConnectByDefaultsFile(defaultsFile)
		}
	}

	return connector
}
