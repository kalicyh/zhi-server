package zhipu

import (
	"fmt"
	"time"
)

func ExampleGenerateVideo(apiKey string) {
	// 创建提供者实例 (默认使用 cogvideox-flash 模型)
	provider := NewZhipuVideoProvider(apiKey,
		WithTimeout(120*time.Second),
	)

	// 创建文生视频请求
	request := &VideoRequest{
		Prompt:    "比得兔开小汽车，游走在马路上，脸上的表情充满开心喜悦。",
		WithAudio: true,         // 开启AI音效生成，支持场景识别
		Size:      ResolutionHD, // 使用1920x1080分辨率
		FPS:       60,           // 使用60fps获得更流畅的效果
	}

	// 发送视频生成请求
	response, err := provider.GenerateVideo(request)
	if err != nil {
		fmt.Printf("生成视频失败: %v\n", err)
		return
	}

	if response.ID == "" {
		fmt.Println("错误：未获取到有效的任务ID")
		return
	}

	fmt.Printf("视频生成任务已提交，任务ID: %s, 状态: %s\n", response.ID, response.TaskStatus)

	// 轮询检查视频生成结果
	maxRetries := 12 // 最大重试次数 (12 * 5秒 = 1分钟)
	retryCount := 0
	time.Sleep(30 * time.Second) // 初始等待30秒
	for retryCount < maxRetries {
		result, err := provider.GetVideoResult(response.ID)
		if err != nil {
			fmt.Printf("获取结果失败: %v\n", err)
			return
		}

		switch result.TaskStatus {
		case "FAIL":
			fmt.Println("视频生成失败")
			return
		case "SUCCESS":
			if len(result.VideoResult) == 0 {
				fmt.Println("视频生成成功但未返回URL")
				return
			}
			for _, v := range result.VideoResult {
				fmt.Printf("视频URL: %s\n", v.URL)
				fmt.Printf("封面URL: %s\n", v.CoverImageURL)
			}
			return
		case "PROCESSING":
			fmt.Println("视频生成中，等待...")
			time.Sleep(5 * time.Second)
			retryCount++
		default:
			fmt.Printf("未知状态: %s\n", result.TaskStatus)
			return
		}
	}

	fmt.Printf("超出最大等待时间(%d秒)，请稍后使用任务ID: %s 查询结果\n",
		maxRetries*5, response.ID)
}

func ExampleGenerateVideoFromImage(apiKey string) {
	// 创建提供者实例 (使用默认的 cogvideox-flash 模型)
	provider := NewZhipuVideoProvider(apiKey)

	// 创建图生视频请求 (支持最长10秒视频)
	request := &VideoRequest{
		ImageURL:  "https://example.com/image.jpg", // 替换为实际的图片URL
		Prompt:    "让画面动起来",
		WithAudio: true,         // 开启AI音效
		Size:      Resolution4K, // 使用4K分辨率 (3840x2160)
		FPS:       60,           // 使用60fps高帧率
	}

	// 发送视频生成请求
	response, err := provider.GenerateVideo(request)
	if err != nil {
		fmt.Printf("生成视频失败: %v\n", err)
		return
	}

	if response.ID == "" {
		fmt.Println("错误：未获取到有效的任务ID")
		return
	}

	fmt.Printf("视频生成任务已提交，任务ID: %s\n", response.ID)

	// 获取结果代码与上面示例相同，建议复用相同的轮询逻辑
}
