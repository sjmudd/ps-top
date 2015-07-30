package connector

import (
	"fmt"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"os"
)

// Flags holds various flags related to connecting to the database
type Flags struct {
	Host         *string
	Socket       *string
	Port         *int
	User         *string
	Password     *string
	DefaultsFile *string
}

// new connector returns a connected Connector given the different parameters
func NewConnector(flags Flags) *Connector {
	var defaultsFile string
	connector := new(Connector)

	if *flags.Host != "" || *flags.Socket != "" {
		logger.Println("--host= or --socket= defined")
		var components = make(map[string]string)
		if *flags.Host != "" && *flags.Socket != "" {
			fmt.Println(lib.MyName() + ": Do not specify --host and --socket together")
			os.Exit(1)
		}
		if *flags.Host != "" {
			components["host"] = *flags.Host
		}
		if *flags.Port != 0 {
			if *flags.Socket == "" {
				components["port"] = fmt.Sprintf("%d", *flags.Port)
			} else {
				fmt.Println(lib.MyName() + ": Do not specify --socket and --port together")
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

	return connector
}
