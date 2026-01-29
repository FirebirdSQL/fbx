package fbconn

import "rdb.red-soft.ru/fbx/fbproto"

type ProtocolFeatures = fbproto.ProtocolFeatures

func ProtocolFeaturesForVersion(version int) ProtocolFeatures {
	return ProtocolFeatures{
		OpExecuteHaveTimeout:     version >= fbproto.PROTOCOL_VERSION16,
		OpExecuteHaveCursorFlags: version >= fbproto.PROTOCOL_VERSION18,
	}
}
