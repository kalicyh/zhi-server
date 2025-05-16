package tts

import (
	"fmt"
	"os"
	"path/filepath"

	"xiaozhi-server-go/src/core/providers"
	"xiaozhi-server-go/src/core/utils"

	"github.com/hajimehoshi/go-mp3"
)

// Config TTS配置结构
type Config struct {
	Type       string                 `yaml:"type"`
	OutputDir  string                 `yaml:"output_dir"`
	Voice      string                 `yaml:"voice,omitempty"`
	Format     string                 `yaml:"format,omitempty"`
	SampleRate int                    `yaml:"sample_rate,omitempty"`
	Extra      map[string]interface{} `yaml:",inline"`
}

// Provider TTS提供者接口
type Provider interface {
	providers.TTSProvider
}

// BaseProvider TTS基础实现
type BaseProvider struct {
	config     *Config
	deleteFile bool
}

// Config 获取配置
func (p *BaseProvider) Config() *Config {
	return p.config
}

// DeleteFile 获取是否删除文件标志
func (p *BaseProvider) DeleteFile() bool {
	return p.deleteFile
}

// NewBaseProvider 创建TTS基础提供者
func NewBaseProvider(config *Config, deleteFile bool) *BaseProvider {
	return &BaseProvider{
		config:     config,
		deleteFile: deleteFile,
	}
}

// Initialize 初始化提供者
func (p *BaseProvider) Initialize() error {
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}
	return nil
}

// Cleanup 清理资源
func (p *BaseProvider) Cleanup() error {
	if p.deleteFile {
		// 清理输出目录中的临时文件
		pattern := filepath.Join(p.config.OutputDir, "*.{wav,mp3,opus}")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("查找临时文件失败: %v", err)
		}
		for _, file := range matches {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("删除临时文件失败: %v", err)
			}
		}
	}
	return nil
}

// Factory TTS工厂函数类型
type Factory func(config *Config, deleteFile bool) (Provider, error)

var (
	factories = make(map[string]Factory)
)

// Register 注册TTS提供者工厂
func Register(name string, factory Factory) {
	factories[name] = factory
}

// Create 创建TTS提供者实例
func Create(name string, config *Config, deleteFile bool) (Provider, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("未知的TTS提供者: %s", name)
	}

	provider, err := factory(config, deleteFile)
	if err != nil {
		return nil, fmt.Errorf("创建TTS提供者失败: %v", err)
	}

	if err := provider.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化TTS提供者失败: %v", err)
	}

	return provider, nil
}

// AudioToOpusData 将音频文件转换为Opus数据块
func (b *BaseProvider) AudioToOpusData(audioFile string) ([][]byte, float64, error) {
	// 先将MP3转为PCM
	pcmData, duration, err := utils.AudioToPCMData(audioFile)
	if err != nil {
		return nil, 0, fmt.Errorf("PCM转换失败: %v", err)
	}

	if len(pcmData) == 0 {
		return nil, 0, fmt.Errorf("PCM转换结果为空")
	}

	// 打开MP3文件获取采样率
	file, err := os.Open(audioFile)
	if err != nil {
		return nil, 0, fmt.Errorf("打开音频文件失败: %v", err)
	}
	defer file.Close()

	// 检查MP3文件格式是否有效
	_, err = mp3.NewDecoder(file)
	if err != nil {
		return nil, 0, fmt.Errorf("创建MP3解码器失败: %v", err)
	}

	// 获取采样率 (固定使用24000Hz作为Opus编码的采样率)
	// 如果采样率不是24000Hz，PCMSlicesToOpusData会处理重采样
	opusSampleRate := 24000
	channels := 1

	// 将PCM转换为Opus
	opusData, err := utils.PCMSlicesToOpusData(pcmData, opusSampleRate, channels, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("PCM转Opus失败: %v", err)
	}

	return opusData, duration, nil
}
