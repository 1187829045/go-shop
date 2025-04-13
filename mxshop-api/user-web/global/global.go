package global

import (
	ut "github.com/go-playground/universal-translator"
	"shop-api/user-web/config"
	"shop-api/user-web/proto"
)

// 全局变量包，用于存储全局配置和对象，方便在整个应用中共享和访问。

var (
	// Trans 是一个全局的翻译器实例，用于处理多语言翻译。
	Trans ut.Translator

	ServerConfig = &config.ServerConfig{}

	NacosConfig = &config.NacosConfig{}

	UserSrvClient proto.UserClient
)
