package fbconn

import "rdb.red-soft.ru/fbx/fbproto"

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
