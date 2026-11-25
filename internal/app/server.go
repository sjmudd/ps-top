package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/tui"
)

// serverVariables and serverStatus are what tui.ServerInfo needs from the
// probe.
var (
	serverVariables = []string{"hostname", "port", "bind_address"}
	serverStatus    = []string{"Uptime"}
)

// serverInfo returns what the TUI's top line shows about the server, from
// the probe, which ran at the time probed.
func serverInfo(srv *probe.Server, probed time.Time) (tui.ServerInfo, error) {
	up, err := strconv.ParseUint(srv.Status["uptime"], 10, 64)
	if err != nil {
		return tui.ServerInfo{}, fmt.Errorf("app: server Uptime %q: %w", srv.Status["uptime"], err)
	}
	return tui.ServerInfo{
		Host:    displayHost(srv.Variables["hostname"], srv.Variables["bind_address"]),
		Port:    srv.Variables["port"],
		Version: srv.Version,
		Started: probed.Add(-time.Duration(up) * time.Second),
	}, nil
}

// displayHost returns bindAddress if it names one address, else hostname.
// Empty (MariaDB's default) and the wildcards mean all interfaces. As the old
// program's display.hostForDisplay.
func displayHost(hostname, bindAddress string) string {
	switch bindAddress {
	case "", "*", "0.0.0.0", "::":
		return hostname
	}
	return bindAddress
}
