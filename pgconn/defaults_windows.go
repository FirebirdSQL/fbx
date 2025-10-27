package pgconn

import (
	"os/user"
	"strings"
)

func defaultSettings() map[string]string {
	settings := make(map[string]string)

	settings["host"] = defaultHost()
	settings["port"] = "3050"
	settings["auth_plugin"] = "Srp"
	settings["wire_crypt"] = "true"

	user, err := user.Current()
	if err == nil {
		username := user.Username
		if strings.Contains(username, "\\") {
			username = username[strings.LastIndex(username, "\\")+1:]
		}

		settings["system_user"] = username
	}

	return settings
}

func defaultHost() string {
	return "localhost"
}
