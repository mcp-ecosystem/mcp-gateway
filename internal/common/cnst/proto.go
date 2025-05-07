package cnst

type ProtoType string

const (
	BackendProtoStdio      ProtoType = "stdio"
	BackendProtoSSE        ProtoType = "sse"
	BackendProtoStreamable ProtoType = "streamable"
	BackendProtoHttp       ProtoType = "http"
	BackendProtoGrpc       ProtoType = "grpc"
)

const (
	FrontendProtoSSE ProtoType = "sse"
)
