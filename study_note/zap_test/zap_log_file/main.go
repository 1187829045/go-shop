package main

import (
	"go.uber.org/zap" // 导入 zap 日志库
	"time"            // 导入时间库
)

// Newlogger 函数用于创建一个新的 zap.Logger 实例
func Newlogger() (*zap.Logger, error) {
	// 创建一个生产环境的 zap 配置
	cfg := zap.NewProductionConfig()

	// 设置日志输出路径，将日志输出到 myproject.log 文件
	cfg.OutputPaths = []string{
		"./myproject.log",
	}

	// 使用配置构建一个新的 logger 实例，并返回
	return cfg.Build()
}

func main() {
	// 使用 Newlogger 函数创建一个新的 logger 实例
	logger, err := Newlogger()
	// 如果创建 logger 过程中发生错误，程序将会 panic
	if err != nil {
		panic(err)
	}

	// 创建一个带有 Suger 的 logger 实例，用于方便的日志记录
	su := logger.Sugar()
	// 确保在 main 函数结束时，将 logger 的缓冲区刷新到日志文件
	defer su.Sync()

	// 定义一个 URL 字符串
	url := "http://imooc.com"

	// 使用 Suger 记录一个 Info 级别的日志消息
	// 包括一些附加字段：url、attempt（尝试次数）、backoff（退避时间）
	su.Info("failed to fetch URL",
		zap.String("url", url),               // 添加字符串类型的字段
		zap.Int("attempt", 3),                // 添加整数类型的字段
		zap.Duration("backoff", time.Second), // 添加时间段类型的字段
	)
}
