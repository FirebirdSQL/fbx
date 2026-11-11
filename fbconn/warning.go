package fbconn

import "github.com/firebirdsql/fbx/fbproto"

type FbWarning struct {
	GdsCodes []int
	Code     string
	Message  string
}

func OpResponseToFbWarning(resp *fbproto.OpResponse) *FbWarning {
	// Copy codes because OpResponse is reused by the client.
	gdsCodes := append([]int(nil), resp.GdsCodes...)
	return &FbWarning{
		GdsCodes: gdsCodes,
		Code:     resp.SqlState,
		Message:  resp.WarnMessage,
	}
}
