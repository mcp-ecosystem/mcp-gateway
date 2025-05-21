package cnst

type ProtoType string

const (
	BackendProtoStdio      ProtoType = "stdio"
	BackendProtoSSE        ProtoType = "sse"
	BackendProtoStreamable ProtoType = "streamable-http"
	BackendProtoHttp       ProtoType = "http"
	BackendProtoGrpc       ProtoType = "grpc"
)

const (
	FrontendProtoSSE ProtoType = "sse"
)

func (s ProtoType) String() string {
	return string(s)
}
