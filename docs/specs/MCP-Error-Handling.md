# MCP 错误响应规范（针对 SSE 和 Streamable HTTP）

## 通用要求

- **HTTP状态码**用于表示请求是否被接受。
- **JSON-RPC 错误对象**用于描述具体错误详情。

标准 JSON-RPC 错误对象格式：

```json
{
  "jsonrpc": "2.0",
  "id": "与请求对应的ID或null",
  "error": {
    "code": 错误码,
    "message": "错误简要描述",
    "data": 可选的附加信息
  }
}
```

常用错误码（遵循 JSON-RPC）：

| 错误码  | 含义             |
|-------|----------------|
| -32700 | 解析错误           |
| -32600 | 无效请求           |
| -32601 | 方法未找到         |
| -32602 | 无效参数           |
| -32603 | 内部错误           |
| -32000~-32099 | 服务器自定义错误 |

---

## SSE 模式下错误处理

- **请求格式错误** ➔ `400 Bad Request`，可附错误JSON体，无`id`。
- **仅发送通知** ➔ `202 Accepted`，无正文。
- **正常请求** ➔ `200 OK`，开启 `Content-Type: text/event-stream` 流。
    - 每个请求对应至少一个响应（成功或错误）。
    - 错误通过 SSE `data:` 发送标准 JSON-RPC 错误对象。

示例 SSE 错误事件：

```text
data: {"jsonrpc":"2.0","id":"123","error":{"code":-32601,"message":"Method not found"}}
```

- **SSE连接不支持** ➔ `405 Method Not Allowed`。

---

## Streamable HTTP 模式下错误处理

- **请求格式错误** ➔ `400 Bad Request`。
- **仅发送通知** ➔ `202 Accepted`。
- **成功请求返回** ➔ `200 OK`，直接返回JSON对象或数组。
    - 单个请求出错 ➔ 响应体为错误对象。
    - 批量请求部分出错 ➔ 响应体是数组，出错条目含`error`字段。
- **Accept: text/event-stream** ➔ 返回 SSE 流，与 SSE 模式规则一致。

---

## 小结

- HTTP层状态码表示请求是否被接受。
- JSON-RPC层错误对象提供具体错误信息。
- 每个请求，无论成功或失败，都必须有对应响应。

## 参考
- https://modelcontextprotocol.io/specification/2025-03-26/server/tools#error-handling
- https://modelcontextprotocol.io/docs/concepts/architecture#error-handling
