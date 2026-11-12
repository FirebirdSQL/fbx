package fbconn

import (
	"os"
	"os/user"
	"runtime/debug"
	"strings"

	"github.com/firebirdsql/fbx/fbproto/auth"
)

// currentOSUser resolves the current OS user account. It is a seam so tests
// can simulate os/user.Current failing -- including the unrecoverable
// crashes reported in some restricted/broken-NSS container environments --
// without needing such an environment.
var currentOSUser = user.Current

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

// osUserSettings returns the default system_user derived from the current
// OS account name. This is independent of the database user.
//
// Resolving this requires looking up the OS user account, which can be slow
// or, in some restricted container environments (e.g. distroless images
// with a broken NSS/CGO setup), crash the process. Callers must only call
// osUserSettings when system_user is empty after parsing the connection string.
func osUserSettings() map[string]string {
	settings := make(map[string]string)

	// Ignore lookup errors and leave system_user without a default. Applications
	// can supply system_user explicitly in the connection string.
	user, err := currentOSUser()
	if err == nil {
		// Windows gives us the username here as `DOMAIN\user` or `LOCALPCNAME\user`,
		// but we strip off the first part.
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
