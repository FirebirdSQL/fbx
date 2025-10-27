//go:build !windows

package pgconn

import "os/user"

func defaultSettings() map[string]string {
	settings := make(map[string]string)

	settings["host"] = defaultHost()
	settings["port"] = "3050"
	settings["auth_plugin"] = "Srp"
	settings["wire_crypt"] = "true"

	user, err := user.Current()
	if err == nil {
		settings["system_user"] = user.Username
	}
	return settings
}

func defaultHost() string {
	return "localhost"
}
