package fbconn

import "os/user"

// SetOSUserLookupForTest replaces the lookup used for the default system_user.
// Tests using it must not run in parallel.
func SetOSUserLookupForTest(lookup func() (*user.User, error)) (restore func()) {
	prev := currentOSUser
	currentOSUser = lookup
	return func() { currentOSUser = prev }
}
