package tts

import (
	"fmt"
	"os"
	"path/filepath"

	"xiaozhi-server-go/src/core/providers"
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

func (bp *BaseProvider) AudioToOpusData(audioFile string) ([][]byte, float64, error) {
	/**
	file, err := os.Open(audioFile)
	if err != nil {
		return nil, 0, fmt.Errorf("打开音频文件失败: %v", err)
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return nil, 0, fmt.Errorf("创建MP3解码器失败: %v", err)
	}

	mp3SampleRate := decoder.SampleRate()

	// 检查采样率是否支持
	supportedRates := map[int]bool{8000: true, 12000: true, 16000: true, 24000: true, 48000: true}
	if !supportedRates[mp3SampleRate] {
		return nil, 0, fmt.Errorf("MP3采样率 %dHz 不被Opus直接支持，需要重采样", mp3SampleRate)
	}

	pcmBytes := make([]byte, decoder.Length())
	if _, err := io.ReadFull(decoder, pcmBytes); err != nil {
		return nil, 0, fmt.Errorf("读取PCM数据失败: %v", err)
	}

	// go-mp3 解码为 16-bit little-endian stereo PCM
	// 将 stereo []byte 转换为 mono []int16
	numStereoSamples := len(pcmBytes) / 2 // 每个样本2字节
	if numStereoSamples == 0 {
		return [][]byte{}, 0, nil // 空音频
	}
	numMonoSamples := numStereoSamples / 2
	pcmMonoInt16 := make([]int16, numMonoSamples)

	for i := 0; i < numMonoSamples; i++ {
		leftSample := int16(pcmBytes[i*4+0]) | (int16(pcmBytes[i*4+1]) << 8)
		rightSample := int16(pcmBytes[i*4+2]) | (int16(pcmBytes[i*4+3]) << 8)
		// 混合为单声道 (简单平均)
		pcmMonoInt16[i] = int16((int32(leftSample) + int32(rightSample)) / 2)
	}

	// 创建Opus编码器参数
	params := opuspkg.NewOpusParams(mp3SampleRate, 1, 20) // 20ms帧长
	encoder, err := opuspkg.NewEncoder(params)
	if err != nil {
		return nil, 0, fmt.Errorf("创建Opus编码器失败: %v", err)
	}
	defer encoder.Close()

	// 准备PCM数据供编码
	samplesPerFrame := (mp3SampleRate * 20) / 1000 // 20ms帧长
	var opusPackets [][]byte

	for i := 0; i < len(pcmMonoInt16); i += samplesPerFrame {
		end := i + samplesPerFrame
		var pcmFrame []byte

		if end > len(pcmMonoInt16) {
			// 处理最后一个不完整帧，用静音填充
			frame := make([]int16, samplesPerFrame)
			copy(frame, pcmMonoInt16[i:])
			// 转换为字节切片
			pcmFrame = make([]byte, samplesPerFrame*2)
			for j, sample := range frame {
				pcmFrame[j*2] = byte(sample)
				pcmFrame[j*2+1] = byte(sample >> 8)
			}
		} else {
			// 转换完整帧为字节切片
			pcmFrame = make([]byte, samplesPerFrame*2)
			for j, sample := range pcmMonoInt16[i:end] {
				pcmFrame[j*2] = byte(sample)
				pcmFrame[j*2+1] = byte(sample >> 8)
			}
		}

		// 编码PCM帧
		encoded, err := encoder.Encode(pcmFrame)
		if err != nil {
			return nil, 0, fmt.Errorf("Opus编码失败: %v", err)
		}

		opusPackets = append(opusPackets, encoded)
	}

	duration := float64(len(pcmMonoInt16)) / float64(mp3SampleRate)
	return opusPackets, duration, nil
	*/
	return nil, 0, fmt.Errorf("音频转Opus格式的功能尚未实现")
}
