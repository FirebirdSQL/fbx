//go:build !windows

package fbconn

import (
	"os"
	"os/user"

	"github.com/red-soft/fbx/fbproto/auth"
)

func defaultSettings() map[string]string {
	settings := make(map[string]string)

	settings["host"] = defaultHost()
	settings["port"] = "3050"
	settings["auth_plugin_list"] = auth.DefaultPluginListString()
	settings["wire_crypt"] = "true"

	user, err := user.Current()
	if err == nil {
		settings["system_user"] = user.Username
	}

	hostname, err := os.Hostname()
	if err == nil {
		settings["system_host"] = hostname
	}

	return settings
}

func defaultHost() string {
	return "localhost"
}
