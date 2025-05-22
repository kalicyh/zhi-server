package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core"
	"xiaozhi-server-go/src/core/utils"

	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

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

	// 创建 WebSocket 服务
	wsServer, err := core.NewWebSocketServer(config, logger)
	if err != nil {
		logger.Error("创建 WebSocket 服务器失败", err)
		os.Exit(1)
	}

	// 初始化Gin引擎
	if config.Log.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// 静态文件 & SPA 回退
	router.Static("/", filepath.Join("web", "dist"))
	router.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join("web", "dist", "index.html"))
	})

	// 用 errgroup 管理两个服务
	g, ctx := errgroup.WithContext(context.Background())

	// HTTP Server（支持优雅关机）
	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(config.Web.Port),
		Handler: router,
	}
	g.Go(func() error {
		logger.Info(fmt.Sprintf("Gin 服务已启动，访问地址: http://localhost:%d", config.Web.Port))
		// ListenAndServe 返回 ErrServerClosed 时表示正常关闭
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP 服务启动失败", err)
			return err
		}
		return nil
	})

	// WebSocket 服务
	g.Go(func() error {
		if err := wsServer.Start(ctx); err != nil {
			logger.Error("WebSocket 服务运行失败", err)
			return err
		}
		return nil
	})

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("接收到系统信号，准备关闭服务", sig)
	case <-ctx.Done():
		logger.Info("服务上下文已取消，准备关闭服务")
	}

	// 触发优雅关机：先给 HTTP Server 5s 时间，再停止 WS
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP 服务优雅关机失败", err)
	} else {
		logger.Info("HTTP 服务已优雅关闭")
	}

	if err := wsServer.Stop(); err != nil {
		logger.Error("WebSocket 服务关闭失败", err)
	} else {
		logger.Info("WebSocket 服务已关闭")
	}

	// 等待 errgroup 中其他 goroutine 退出
	if err := g.Wait(); err != nil {
		logger.Error("服务退出时出现错误", err)
		// logger.Close()
		os.Exit(1)
	}

	logger.Info("所有服务已成功关闭，程序退出")
	// logger.Close()
}
