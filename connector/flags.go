package connector

import (
	"fmt"
	"log"
	"os"

	"github.com/sjmudd/ps-top/lib"
)

// Flags holds various command line flags related to connecting to the database
type Flags struct {
	Host           *string // the host to connect to
	Socket         *string // the unix socket to connect with
	Port           *int    // the port to connect to
	User           *string // the user to connect with
	Password       *string // the password to use
	DefaultsFile   *string // name of the defaults file to use
	UseEnvironment *bool   // use the environment to set connection settings?
}

// NewConnector returns a connected Connector given the provided flags
func NewConnector(flags Flags) *Connector {
	var defaultsFile string
	connector := new(Connector)

	if *flags.UseEnvironment {
		connector.ConnectByEnvironment()
	} else {
		if *flags.Host != "" || *flags.Socket != "" {
			log.Println("--host= or --socket= defined")
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
