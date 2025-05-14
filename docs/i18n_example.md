# I18n 国际化错误处理示例

本文档展示了如何在MCP Gateway中使用新的国际化错误处理机制。

## 基本用法

新的国际化错误处理机制使用了自定义错误类型`I18nError`，它可以包含一个消息ID和可选的数据参数。中间件会自动检测这些错误并将其翻译为适当的语言。

### 在路由处理器中返回错误

```go
// 导入必要的包
import (
    "github.com/gin-gonic/gin"
    "github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/middleware"
    "github.com/mcp-ecosystem/mcp-gateway/pkg/i18n"
)

// 简单的错误
func GetUser(c *gin.Context) {
    userID := c.Param("id")
    user, err := userService.GetByID(userID)
    if err != nil {
        // 返回一个国际化错误
        c.Error(middleware.GetI18nError("ErrorUserNotFound"))
        return
    }
    c.JSON(200, user)
}

// 带参数的错误
func CreateUser(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        // 返回带参数的国际化错误
        c.Error(middleware.GetI18nErrorWithData("ErrorInvalidInput", map[string]interface{}{
            "Field": "name",
            "Reason": "Cannot be empty",
        }))
        return
    }
    
    // 继续处理...
}

// 带状态码的错误
func DeleteUser(c *gin.Context) {
    userID := c.Param("id")
    if !hasPermission(c, userID) {
        // 返回带状态码的国际化错误
        c.Error(middleware.GetI18nErrorWithCode("ErrorForbidden", i18n.ErrorForbidden))
        return
    }
    
    err := userService.Delete(userID)
    if err != nil {
        if isNotFound(err) {
            c.Error(middleware.GetI18nErrorWithCode("ErrorUserNotFound", i18n.ErrorNotFound))
        } else {
            c.Error(middleware.GetI18nErrorWithCode("ErrorInternalServer", i18n.ErrorInternalServer))
        }
        return
    }
    
    c.Status(204)
}

// 带状态码和参数的错误
func UpdateUser(c *gin.Context) {
    userID := c.Param("id")
    var updates UserUpdates
    
    if err := c.ShouldBindJSON(&updates); err != nil {
        c.Error(middleware.GetI18nErrorWithCodeAndData(
            "ErrorValidationFailed", 
            i18n.ErrorBadRequest,
            map[string]interface{}{"Reason": err.Error()},
        ))
        return
    }
    
    // 继续处理...
}
```

## 错误类型

新的国际化错误处理机制支持以下几种错误类型：

1. **基本错误** - 使用`GetI18nError("ErrorMessageID")`
2. **带参数的错误** - 使用`GetI18nErrorWithData("ErrorMessageID", dataMap)`
3. **带状态码的错误** - 使用`GetI18nErrorWithCode("ErrorMessageID", code)`
4. **带状态码和参数的错误** - 使用`GetI18nErrorWithCodeAndData("ErrorMessageID", code, dataMap)`

## 预定义错误

`pkg/i18n`包中预定义了一些常用的错误：

```go
// 预定义错误
var (
    ErrNotFound = NewErrorWithCode("ErrorResourceNotFound", ErrorNotFound)
    ErrUnauthorized = NewErrorWithCode("ErrorUnauthorized", ErrorUnauthorized)
    ErrForbidden = NewErrorWithCode("ErrorForbidden", ErrorForbidden)
    ErrBadRequest = NewErrorWithCode("ErrorBadRequest", ErrorBadRequest)
    ErrInternalServer = NewErrorWithCode("ErrorInternalServer", ErrorInternalServer)
)
```

你可以直接使用这些预定义错误：

```go
func GetResource(c *gin.Context) {
    id := c.Param("id")
    resource, err := resourceService.GetByID(id)
    if err != nil {
        if isNotFound(err) {
            // 使用预定义错误
            c.Error(i18n.ErrNotFound.WithParam("ID", id))
            return
        }
        c.Error(i18n.ErrInternalServer)
        return
    }
    
    c.JSON(200, resource)
}
```

## 翻译文件配置

确保在翻译文件中定义了相应的消息ID：

```toml
# translations/en/messages.toml
[ErrorUserNotFound]
other = "User not found"

[ErrorInvalidInput]
other = "Invalid input: field {{.Field}} {{.Reason}}"

[ErrorForbidden]
other = "You do not have permission to perform this action"

[ErrorValidationFailed]
other = "Validation failed: {{.Reason}}"

[ErrorResourceNotFound]
other = "Resource with ID {{.ID}} not found"
```

```toml
# translations/zh/messages.toml
[ErrorUserNotFound]
other = "找不到用户"

[ErrorInvalidInput]
other = "无效输入：字段 {{.Field}} {{.Reason}}"

[ErrorForbidden]
other = "您没有权限执行此操作"

[ErrorValidationFailed]
other = "验证失败：{{.Reason}}"

[ErrorResourceNotFound]
other = "找不到ID为 {{.ID}} 的资源"
```

## 工作原理

新的国际化错误处理机制的工作原理如下：

1. 处理程序返回一个`I18nError`或`ErrorWithCode`类型的错误
2. 中间件检测到这些错误，获取请求中的语言偏好
3. 中间件使用翻译器将错误消息ID翻译为适当的语言
4. 中间件将翻译后的错误消息发送给客户端

这种方法比以前的方法更高效，因为它不需要解析和修改整个响应体，而是在错误处理的早期阶段就进行翻译。 