package main

import (
	"fmt"
	"go-redis/config"
	"go-redis/lib/logger"
	"go-redis/tcp"
	"os"
)

// 配置文件名称
const configFile string = "redis.conf"

// 初始化配置参数
var defaultProperties = &config.ServerProperties{
	Bind: "0.0.0.0",
	Port: 6379,
}

func fileExists(fileName string) bool {
	info, err := os.Stat(fileName)
	return err == nil && !info.IsDir()
}

func main() {
	// 日志模块设置
	logger.Setup(&logger.Settings{
		Path:       "logs",
		Name:       "godis",
		Ext:        "log",
		TimeFormat: "2006-01-02",
	})

	if fileExists(configFile) {
		config.SetupConfig(configFile)
	} else {
		config.Properties = defaultProperties
	}

	// 把 tcp 服务启动起来
	err := tcp.ListenAndServeWithSignal(
		&tcp.Config{Address: fmt.Sprintf("%s:%d", config.Properties.Bind, config.Properties.Port)},
		tcp.NewEchoHandler(),
	)
	if err != nil {
		logger.Error(err)
	}

	//git remote set-url master https://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@github.com/chenglh/goredis.git

}
