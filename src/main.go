package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core"
	"xiaozhi-server-go/src/core/utils"

	"strconv"

	"github.com/gin-gonic/gin"

	// 导入所有providers以确保init函数被调用
	_ "xiaozhi-server-go/src/core/providers/asr/doubao"
	_ "xiaozhi-server-go/src/core/providers/llm/openai"
	_ "xiaozhi-server-go/src/core/providers/tts/doubao"
	_ "xiaozhi-server-go/src/core/providers/tts/edge"
)

func main() {
	// 加载配置,默认使用src/configs/.config.yaml
	config, configPath, err := configs.LoadConfig()
	if err != nil {
		panic(err)
	}

	// 初始化日志系统
	logger, err := utils.NewLogger(config)
	if err != nil {
		panic(err)
	}
	logger.Info(fmt.Sprintf("日志系统初始化成功, 配置文件路径: %s", configPath))
	if configPath == "config.yaml" {
		logger.Warn("推荐使用本地配置文件src/configs/.config.yaml, 避免关键信息泄露")
	}
	defer logger.Close()

	// 创建WebSocket服务器
	server, err := core.NewWebSocketServer(config, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("创建WebSocket服务器失败: %v", err))
		os.Exit(1)
	}

	// 初始化Gin引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 设置可信代理（根据部署环境配置）
	router.SetTrustedProxies([]string{"127.0.0.1"}) // 开发环境本地代理

	// 静态文件服务
	router.Static("/", filepath.Join("web", "dist"))

	// 前端路由回退
	router.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join("web", "dist", "index.html"))
	})

	// 创建上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动Gin服务器
	go func() {
		logger.Info(fmt.Sprintf("Gin服务器启动成功 监听端口:%d 访问地址: http://localhost:%d 运行模式:%s", config.Web.Port, config.Web.Port, gin.Mode()))
		if err := router.Run(":" + strconv.Itoa(config.Web.Port)); err != nil {
			logger.Error("Gin服务器启动失败", err)
			cancel()
		}
	}()

	// 启动WebSocket服务器
	go func() {
		if err := server.Start(ctx); err != nil {
			logger.Error("WebSocket服务器运行失败", err)
			cancel()
		}
	}()

	// 等待信号
	select {
	case sig := <-sigChan:
		logger.Info("接收到信号，准备关闭服务器", sig)
	case <-ctx.Done():
		logger.Info("服务器上下文已取消")
	}

	// 优雅关闭
	if err := server.Stop(); err != nil {
		logger.Error("服务器关闭失败", err)
		os.Exit(1)
	}

	logger.Info("服务器已成功关闭")
}
