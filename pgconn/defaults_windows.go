package pgconn

import (
	"os"
	"os/user"
	"strings"
)

func defaultSettings() map[string]string {
	settings := make(map[string]string)

	settings["host"] = defaultHost()
	settings["port"] = "3050"

	user, err := user.Current()
	appData := os.Getenv("APPDATA")
	if err == nil {
		username := user.Username
		if strings.Contains(username, "\\") {
			username = username[strings.LastIndex(username, "\\")+1:]
		}

		settings["user"] = username
	}

	return settings
}

func defaultHost() string {
	return "localhost"
}
