package fbconn

import "github.com/red-soft/fbx/fbproto"

type FbWarning struct {
	GdsCodes []int
	Code     string
	Message  string
}

func OpResponseToFbWarning(resp *fbproto.OpResponse) *FbWarning {
	return &FbWarning{
		GdsCodes: resp.GdsCodes,
		Code:     resp.SqlState,
		Message:  resp.WarnMessage,
	}
}
