package fbconn

import "rdb.red-soft.ru/fbx/fbproto"

// ProtocolFeatures aliases fbproto.ProtocolFeatures for use in fbconn.
type ProtocolFeatures = fbproto.ProtocolFeatures

// ProtocolFeaturesForVersion builds protocol feature flags for a wire version.
func ProtocolFeaturesForVersion(version int) ProtocolFeatures {
	return ProtocolFeatures{
		OpExecuteHaveTimeout:         version >= fbproto.PROTOCOL_VERSION16,
		OpExecuteHaveCursorFlags:     version >= fbproto.PROTOCOL_VERSION18,
		OpExecuteHaveBlobInlineSize:  version >= fbproto.PROTOCOL_VERSION19,
		OpExecuteHaveBlobInlineCache: version >= fbproto.PROTOCOL_VERSION19,
	}
}
