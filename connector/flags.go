package connector

import (
	"fmt"
	"github.com/sjmudd/ps-top/lib"
	"github.com/sjmudd/ps-top/logger"
	"os"
)

// new connector returns a connected Connector given the different parameters
func NewConnector(flagHost, flagSocket *string, flagPort *int, flagUser, flagPassword, flagDefaultsFile *string) *Connector {
	var defaultsFile string
	connector := new(Connector)

	if *flagHost != "" || *flagSocket != "" {
		logger.Println("--host= or --socket= defined")
		var components = make(map[string]string)
		if *flagHost != "" && *flagSocket != "" {
			fmt.Println(lib.MyName() + ": Do not specify --host and --socket together")
			os.Exit(1)
		}
		if *flagHost != "" {
			components["host"] = *flagHost
		}
		if *flagPort != 0 {
			if *flagSocket == "" {
				components["port"] = fmt.Sprintf("%d", *flagPort)
			} else {
				fmt.Println(lib.MyName() + ": Do not specify --socket and --port together")
				os.Exit(1)
			}
		}
		if *flagSocket != "" {
			components["socket"] = *flagSocket
		}
		if *flagUser != "" {
			components["user"] = *flagUser
		}
		if *flagPassword != "" {
			components["password"] = *flagPassword
		}
		connector.ConnectByComponents(components)
	} else {
		if flagDefaultsFile != nil && *flagDefaultsFile != "" {
			logger.Println("--defaults-file defined")
			defaultsFile = *flagDefaultsFile
		} else {
			logger.Println("connecting by implicit defaults file")
		}
		connector.ConnectByDefaultsFile(defaultsFile)
	}

	return connector
}
