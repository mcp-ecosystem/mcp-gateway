package cnst

type ProtoType string

const (
	BackendProtoStdio ProtoType = "stdio"
	BackendProtoSSE   ProtoType = "sse"
	BackendProtoHttp  ProtoType = "http"
	BackendProtoGrpc  ProtoType = "grpc"
)

const (
	FrontendProtoSSE ProtoType = "sse"
)
