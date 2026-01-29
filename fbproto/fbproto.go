package fbproto

type ClientMessage interface {
	Write(x *XDRWriter) error
}

type ServerMessage interface {
	Read(x *XDRReader) error
}

type ProtocolVersionDescriptor struct {
	Version  int
	ArchType int
	MinType  int
	MaxType  int
	Weight   int
}

var (
	SupportedProtocols = []ProtocolVersionDescriptor{
		{PROTOCOL_VERSION13 | ProtocolFlag, GenericAchitectureClient, Ptype_lazy_send, Ptype_lazy_send, 8},
		{PROTOCOL_VERSION14 | ProtocolFlag, GenericAchitectureClient, Ptype_lazy_send, Ptype_lazy_send, 10},
		{PROTOCOL_VERSION15 | ProtocolFlag, GenericAchitectureClient, Ptype_lazy_send, Ptype_lazy_send, 12},
		{PROTOCOL_VERSION16 | ProtocolFlag, GenericAchitectureClient, Ptype_lazy_send, Ptype_lazy_send, 14},
		{PROTOCOL_VERSION17 | ProtocolFlag, GenericAchitectureClient, Ptype_lazy_send, Ptype_lazy_send, 16},
		{PROTOCOL_VERSION18 | ProtocolFlag, GenericAchitectureClient, Ptype_lazy_send, Ptype_lazy_send, 18},
	}
)
