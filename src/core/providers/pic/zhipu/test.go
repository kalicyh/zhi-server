package zhipu

import (
    "fmt"
    "log"
    "time"
)

// TestZhipuImageGeneration 测试智普文生图功能
func TestZhipuImageGeneration(apiKey string) {

    fmt.Println("开始测试智普文生图功能...")

    // 初始化智普提供者
    provider := NewZhipuProvider(
        apiKey,
        WithModel("cogview-3-flash"), // 使用默认模型
        WithTimeout(60*time.Second),
    )

    // 调用文生图接口
    prompt := "一只可爱的橘色猫咪在窗台上晒太阳，背景是蓝天白云"
    fmt.Printf("图像描述: %s\n", prompt)
    
    response, err := provider.GenerateImage(
        prompt,        // 图像描述
        "standard",    // 质量选项: standard 或 hd
        "1024x1024",   // 图片尺寸
        "test_user",   // 用户ID (可选)
    )

    if err != nil {
        log.Fatalf("生成图片失败: %v", err)
    }

    // 输出生成的图片URL
    fmt.Println("图片生成成功!")
    fmt.Printf("图片URL: %s\n", response.Data[0].URL)
    fmt.Println("注意: 图片链接有效期为30天，请及时保存")
    
    // 输出内容过滤信息（如果有）
    if len(response.ContentFilter) > 0 {
        fmt.Println("\n内容过滤信息:")
        for _, filter := range response.ContentFilter {
            fmt.Printf("- 角色: %s, 级别: %d\n", filter.Role, filter.Level)
        }
    }
}