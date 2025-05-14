# 国际化错误处理总结

我们已经实现了一个新的国际化错误处理系统，它允许直接在错误发生点创建和翻译错误消息，而不需要通过中间件解析和修改响应体。

## 主要组件

1. **I18nError类型** - 定义在`pkg/i18n/error.go`中，包含消息ID、默认消息和模板数据
2. **ErrorWithCode类型** - 扩展I18nError，添加了HTTP状态码支持
3. **全局翻译器** - 在应用程序启动时初始化，用于所有国际化消息的翻译
4. **辅助函数** - 简化错误创建和处理的函数，如`RespondWithError`和`TranslateMessageGin`

## 用法示例

### 创建国际化错误

```go
// 使用预定义错误
return i18n.ErrNotFound.WithParam("ID", id)

// 创建自定义错误
return i18n.NewErrorWithCode("ErrorTenantNotFound", i18n.ErrorNotFound).WithParam("Name", tenantName)
```

### 在HTTP处理程序中使用

```go
func GetResource(c *gin.Context) {
    id := c.Param("id")
    resource, err := resourceService.GetByID(id)
    if err != nil {
        // 简单方式：使用辅助函数发送错误响应
        i18n.RespondWithError(c, i18n.ErrNotFound.WithParam("ID", id))
        return
    }
    
    // 翻译成功消息
    c.JSON(http.StatusOK, gin.H{
        "message": i18n.TranslateMessageGin("SuccessResourceFound", c, nil),
        "data": resource,
    })
}
```

## 工作原理

1. 当一个请求到达服务器时，`I18nMiddleware`中间件会提取并存储用户的语言首选项
2. 当需要返回一个错误时，直接创建一个`I18nError`或使用预定义错误
3. 使用`RespondWithError`发送带有适当状态码和翻译错误消息的HTTP响应
4. 使用`TranslateMessageGin`翻译成功消息或其他消息字符串

## 优势

1. **简单直接** - 不需要复杂的中间件逻辑来解析和修改响应
2. **更好的类型安全** - 使用专用错误类型而不是字符串
3. **更好的语义** - 错误定义在它们发生的地方
4. **一致的接口** - 辅助函数提供一致的错误处理接口

## 预定义错误

```go
var (
    ErrNotFound = NewErrorWithCode("ErrorResourceNotFound", ErrorNotFound)
    ErrUnauthorized = NewErrorWithCode("ErrorUnauthorized", ErrorUnauthorized)
    ErrForbidden = NewErrorWithCode("ErrorForbidden", ErrorForbidden)
    ErrBadRequest = NewErrorWithCode("ErrorBadRequest", ErrorBadRequest)
    ErrInternalServer = NewErrorWithCode("ErrorInternalServer", ErrorInternalServer)
)
```

## 翻译文件

翻译文件位于`configs/i18n/{lang}/messages.toml`，使用以下格式：

```toml
[ErrorTenantNotFound]
other = "Tenant with name '{{.Name}}' not found"
```

## 进一步改进

1. 添加更多预定义错误
2. 为特定模块创建专用错误
3. 添加日志记录和错误追踪功能
4. 考虑添加错误分类和分组支持 