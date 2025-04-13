package router

import (
	"github.com/gin-gonic/gin"      // 导入 Gin 框架
	"shop-api/user-web/api"         // 导入用户相关的 API 处理函数
	"shop-api/user-web/middlewares" // 导入中间件包
)

// InitUserRouter 初始化用户相关的路由
func InitUserRouter(Router *gin.RouterGroup) {
	UserRouter := Router.Group("user") // 在传入的 RouterGroup 上创建一个 "user" 的路由组
	{
		// GET 请求：获取用户列表，使用 JWTAuth 和 IsAdminAuth 中间件进行身份验证和权限控制，处理函数为 api.GetUserList
		UserRouter.GET("", middlewares.JWTAuth(), middlewares.IsAdminAuth(), api.GetUserList)

		// POST 请求：用户密码登录，处理函数为 api.PassWordLogin
		UserRouter.POST("pwd_login", api.PassWordLogin)

		// POST 请求：用户注册，处理函数为 api.Register
		UserRouter.POST("register", api.Register)

	}
}
