# 直接使用国际化错误

本文档展示了如何直接使用国际化错误处理机制，而不需要通过中间件。

## 基本原理

我们可以直接在错误发生点使用带有国际化支持的错误类型，这样就可以直接返回给客户端一个易于翻译的错误消息。当客户端指定了语言首选项（通过HTTP头`X-Lang`或`Accept-Language`），后端会自动根据这些首选项将错误消息翻译为适当的语言。

## 初始化全局翻译器

在应用程序启动时，应该初始化全局翻译器：

```go
func main() {
    // 初始化翻译器
    if err := i18n.InitTranslator("configs/i18n"); err != nil {
        log.Printf("Warning: Failed to load translations: %v\n", err)
    }
    
    // 设置默认语言（可选，默认为zh）
    i18n.SetDefaultLanguage("zh")
    
    // ...其他初始化代码
}
```

## 用法示例

### 引入必要的包

```go
import (
    "github.com/mcp-ecosystem/mcp-gateway/pkg/i18n"
)
```

### 使用预定义错误

```go
// 返回一个预定义的错误
if user == nil {
    return nil, i18n.ErrNotFound
}

// 返回带参数的预定义错误
if !hasPermission() {
    return nil, i18n.ErrForbidden.WithParam("Resource", resourceName)
}
```

### 创建自定义错误

```go
// 创建一个新的国际化错误
if tenant == nil {
    return nil, i18n.NewErrorWithCode("ErrorTenantNotFound", i18n.ErrorNotFound).WithParam("Name", tenantName)
}

// 带参数的错误
if err := validate(input); err != nil {
    return nil, i18n.NewErrorWithCode("ErrorValidationFailed", i18n.ErrorBadRequest).WithParam("Reason", err.Error())
}
```

### 在HTTP处理程序中使用辅助函数

我们提供了一些辅助函数，使得在HTTP处理程序中更容易使用国际化错误：

```go
func GetUser(c *gin.Context) {
    userID := c.Param("id")
    user, err := userService.GetByID(userID)
    if err != nil {
        // 使用辅助函数处理错误
        i18n.RespondWithError(c, err)
        return
    }
    
    c.JSON(http.StatusOK, user)
}
```

### 翻译成功消息

```go
// 翻译成功消息
c.JSON(http.StatusOK, gin.H{
    "message": i18n.TranslateMessageGin("SuccessResourceCreated", c, nil),
    "data": result,
})

// 带参数的成功消息
c.JSON(http.StatusOK, gin.H{
    "message": i18n.TranslateMessageGin("SuccessResourceCreated", c, map[string]interface{}{
        "ResourceType": "User",
        "ResourceName": user.Name,
    }),
    "data": user,
})
```

## 完整示例

下面是一个完整的HTTP处理程序示例：

```go
func CreateTenant(c *gin.Context) {
    var req dto.CreateTenantRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
        return
    }

    // Validate request
    if req.Name == "" || req.Prefix == "" {
        i18n.RespondWithError(c, i18n.NewErrorWithCode("ErrorTenantRequiredFields", i18n.ErrorBadRequest))
        return
    }

    // Create tenant
    tenant, err := tenantService.Create(c.Request.Context(), req)
    if err != nil {
        if isDuplicateName(err) {
            i18n.RespondWithError(c, i18n.NewErrorWithCode("ErrorTenantNameExists", i18n.ErrorConflict))
        } else if isDuplicatePrefix(err) {
            i18n.RespondWithError(c, i18n.NewErrorWithCode("ErrorTenantPrefixExists", i18n.ErrorConflict))
        } else {
            i18n.RespondWithError(c, i18n.ErrInternalServer)
        }
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "id": tenant.ID,
        "message": i18n.TranslateMessageGin("SuccessTenantCreated", c, nil),
    })
}
```

## 如何工作

该机制的工作原理如下：

1. 全局翻译器在应用程序启动时加载所有翻译文件
2. `I18nError`和`ErrorWithCode`类型包含消息ID和可选的数据参数
3. 当需要翻译错误时，系统从HTTP请求中提取语言首选项
4. 使用翻译器将消息ID翻译为适当的语言
5. 辅助函数`RespondWithError`和`TranslateMessageGin`简化了与Gin的集成

## 主要功能和辅助函数

包`pkg/i18n`提供以下主要功能：

- `InitTranslator(path string)` - 初始化全局翻译器
- `SetDefaultLanguage(lang string)` - 设置默认语言
- `New(messageID string)` - 创建新的I18nError
- `NewErrorWithCode(messageID string, code ErrorCode)` - 创建带状态码的错误
- `RespondWithError(c *gin.Context, err error)` - 发送错误响应
- `TranslateMessageGin(msgID string, c *gin.Context, data map[string]interface{})` - 翻译消息

## 预定义错误

`pkg/i18n`包预定义了一些常用错误：

```go
var (
    ErrNotFound = NewErrorWithCode("ErrorResourceNotFound", ErrorNotFound)
    ErrUnauthorized = NewErrorWithCode("ErrorUnauthorized", ErrorUnauthorized)
    ErrForbidden = NewErrorWithCode("ErrorForbidden", ErrorForbidden)
    ErrBadRequest = NewErrorWithCode("ErrorBadRequest", ErrorBadRequest)
    ErrInternalServer = NewErrorWithCode("ErrorInternalServer", ErrorInternalServer)
)
``` 