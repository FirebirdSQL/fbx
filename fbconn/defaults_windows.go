package fbconn

import (
	"os/user"
	"strings"

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
