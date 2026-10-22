package fbconn

import (
	"os"
	"os/user"
	"runtime/debug"
	"strings"

	"rdb.red-soft.ru/fbx/fbproto/auth"
)

func defaultSettings() map[string]string {
	settings := make(map[string]string)

	settings["host"] = defaultHost()
	settings["port"] = defaultPort
	settings["auth_plugin_list"] = auth.DefaultPluginListString()
	settings["wire_crypt"] = "true"
	settings["wire_compression"] = "false"
	settings["encoding"] = "UTF8"
	settings["blob_buffer_size"] = "16384"
	settings["use_stream_blobs"] = "true"
	settings["max_inline_blob_size"] = "65536"
	settings["max_blob_cache_size"] = "10485760"

	user, err := user.Current()
	if err == nil {
		username := user.Username
		if strings.Contains(username, "\\") {
			username = username[strings.LastIndex(username, "\\")+1:]
		}

		settings["system_user"] = username
	}

	hostname, err := os.Hostname()
	if err == nil {
		settings["system_host"] = hostname
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		settings["client_version"] = bi.Main.Version
	}

	settings["target_session_attrs"] = "any"

	return settings
}

func defaultHost() string {
	return "localhost"
}
