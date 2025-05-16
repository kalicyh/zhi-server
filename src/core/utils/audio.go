package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/go-mp3"
)

func MP3ToPCMData(audioFile string) ([][]byte, error) {
	file, err := os.Open(audioFile)
	if err != nil {
		return nil, fmt.Errorf("打开音频文件失败: %v", err)
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return nil, fmt.Errorf("创建MP3解码器失败: %v", err)
	}

	mp3SampleRate := decoder.SampleRate()

	// 检查采样率是否支持
	supportedRates := map[int]bool{8000: true, 12000: true, 16000: true, 24000: true, 48000: true}
	if !supportedRates[mp3SampleRate] {
		return nil, fmt.Errorf("MP3采样率 %dHz 不被Opus直接支持，需要重采样", mp3SampleRate)
	}

	// decoder.Length() 返回解码后的PCM数据总字节数 (16-bit little-endian stereo)
	pcmBytes := make([]byte, decoder.Length())
	// ReadFull确保读取所有请求的字节，否则返回错误
	if _, err := io.ReadFull(decoder, pcmBytes); err != nil {
		// 如果 decoder.Length() 为 0, pcmBytes 为空, ReadFull 读取 0 字节, 返回 nil 错误，这是正常的。
		// 如果 decoder.Length() > 0 且 ReadFull 返回错误, 表示未能读取完整的PCM数据。
		return nil, fmt.Errorf("读取PCM数据失败: %v", err)
	}

	// go-mp3 解码为 16-bit little-endian stereo PCM.
	// pcmBytes 包含交错的立体声数据 (LRLRLR...).
	// 每个立体声样本对 (左16位, 右16位) 占用4字节.
	// numMonoSamples 是转换后得到的16位单声道样本的数量.
	numMonoSamples := len(pcmBytes) / 4

	if numMonoSamples == 0 {
		// 处理 pcmBytes 为空或数据不足以形成一个单声道样本的情况 (即少于4字节).
		return [][]byte{}, nil // 返回空数据
	}

	pcmMonoInt16 := make([]int16, numMonoSamples)
	for i := 0; i < numMonoSamples; i++ {
		// 从pcmBytes中提取16位小端序的左右声道样本
		// pcmBytes[i*4+0] = 左声道低字节, pcmBytes[i*4+1] = 左声道高字节
		// pcmBytes[i*4+2] = 右声道低字节, pcmBytes[i*4+3] = 右声道高字节
		leftSample := int16(uint16(pcmBytes[i*4+0]) | (uint16(pcmBytes[i*4+1]) << 8))
		rightSample := int16(uint16(pcmBytes[i*4+2]) | (uint16(pcmBytes[i*4+3]) << 8))

		// 通过平均值混合为单声道样本
		// 使用int32进行中间求和以防止在除法前溢出
		pcmMonoInt16[i] = int16((int32(leftSample) + int32(rightSample)) / 2)
	}

	// 将 []int16 类型的单声道PCM数据转换为 []byte (仍然是16位小端序)
	monoPcmDataBytes := make([]byte, numMonoSamples*2) // 每个int16样本占用2字节
	for i, sample := range pcmMonoInt16 {
		monoPcmDataBytes[i*2] = byte(sample)        // 低字节 (LSB)
		monoPcmDataBytes[i*2+1] = byte(sample >> 8) // 高字节 (MSB)
	}

	// 函数签名要求返回 [][]byte.
	// 我们将整个单声道PCM数据作为外部切片中的单个段/切片返回.
	return [][]byte{monoPcmDataBytes}, nil
}

func SaveAudioToWavFile(data []byte, fileName string, sampleRate int, channels int, bitsPerSample int) error {
	if fileName == "" {
		fileName = "output.wav"
	}

	isNewFile := false
	fileInfo, err := os.Stat(fileName)

	// 检查文件是否存在
	if os.IsNotExist(err) {
		isNewFile = true
	}

	var file *os.File
	if isNewFile {
		// 创建新文件
		file, err = os.Create(fileName)
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}
		defer file.Close()

		// 写入WAV文件头
		if err := writeWavHeader(file, 0, sampleRate, channels, bitsPerSample); err != nil {
			return fmt.Errorf("写入WAV头失败: %v", err)
		}
	}

	// 打开现有文件进行追加
	file, err = os.OpenFile(fileName, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 获取当前数据大小
	var currentDataSize int64
	if !isNewFile {
		currentDataSize = fileInfo.Size() - 44 // 减去WAV头大小(44字节)
	}

	// 在文件末尾追加新数据
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("定位文件末尾失败: %v", err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("写入数据失败: %v", err)
	}

	// 更新WAV头中的数据大小
	newDataSize := currentDataSize + int64(len(data))
	file.Seek(0, io.SeekStart)
	if err := writeWavHeader(file, int(newDataSize), sampleRate, channels, bitsPerSample); err != nil {
		return fmt.Errorf("更新WAV头失败: %v", err)
	}

	return nil
}

// 写入WAV文件头
func writeWavHeader(file *os.File, dataSize int, sampleRate, channels, bitsPerSample int) error {
	// RIFF块
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))

	// 文件总长度 = 数据大小 + 头部大小(36) - 8
	fileSize := uint32(dataSize + 36)
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)

	// 文件类型
	copy(header[8:12], []byte("WAVE"))

	// 格式块
	copy(header[12:16], []byte("fmt "))

	// 格式块大小(16字节)
	header[16] = 16
	header[17] = 0
	header[18] = 0
	header[19] = 0

	// 音频格式(1表示PCM)
	header[20] = 1
	header[21] = 0

	// 通道数
	header[22] = byte(channels)
	header[23] = 0

	// 采样率
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)

	// 字节率 = 采样率 × 通道数 × 位深度/8
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)

	// 块对齐 = 通道数 × 位深度/8
	blockAlign := uint16(channels * bitsPerSample / 8)
	header[32] = byte(blockAlign)
	header[33] = byte(blockAlign >> 8)

	// 位深度
	header[34] = byte(bitsPerSample)
	header[35] = byte(bitsPerSample >> 8)

	// 数据块
	copy(header[36:40], []byte("data"))

	// 数据大小
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	_, err := file.Write(header)
	return err
}

// 保留原来的函数，但使用新函数
func SaveAudioToFile(data []byte) error {
	// 默认使用16kHz, 单声道, 16位
	return SaveAudioToWavFile(data, "output.wav", 16000, 1, 16)
}

func ReadPCMDataFromWavFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开WAV文件失败: %v", err)
	}
	defer file.Close()

	// 跳过WAV头
	header := make([]byte, 44)
	if _, err := file.Read(header); err != nil {
		return nil, fmt.Errorf("读取WAV头失败: %v", err)
	}

	// 读取PCM数据
	pcmData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取PCM数据失败: %v", err)
	}

	return pcmData, nil
}

func AudioToPCMData(audioFile string) ([][]byte, float64, error) {
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

	// decoder.Length() 返回解码后的PCM数据总字节数 (16-bit little-endian stereo)
	pcmBytes := make([]byte, decoder.Length())
	// ReadFull确保读取所有请求的字节，否则返回错误
	if _, err := io.ReadFull(decoder, pcmBytes); err != nil {
		// 如果 decoder.Length() 为 0, pcmBytes 为空, ReadFull 读取 0 字节, 返回 nil 错误，这是正常的。
		// 如果 decoder.Length() > 0 且 ReadFull 返回错误, 表示未能读取完整的PCM数据。
		return nil, 0, fmt.Errorf("读取PCM数据失败: %v", err)
	}

	// go-mp3 解码为 16-bit little-endian stereo PCM.
	// pcmBytes 包含交错的立体声数据 (LRLRLR...).
	// 每个立体声样本对 (左16位, 右16位) 占用4字节.
	// numMonoSamples 是转换后得到的16位单声道样本的数量.
	numMonoSamples := len(pcmBytes) / 4

	if numMonoSamples == 0 {
		// 处理 pcmBytes 为空或数据不足以形成一个单声道样本的情况 (即少于4字节).
		return [][]byte{}, 0, nil // 返回空数据
	}

	pcmMonoInt16 := make([]int16, numMonoSamples)
	for i := 0; i < numMonoSamples; i++ {
		// 从pcmBytes中提取16位小端序的左右声道样本
		// pcmBytes[i*4+0] = 左声道低字节, pcmBytes[i*4+1] = 左声道高字节
		// pcmBytes[i*4+2] = 右声道低字节, pcmBytes[i*4+3] = 右声道高字节
		leftSample := int16(uint16(pcmBytes[i*4+0]) | (uint16(pcmBytes[i*4+1]) << 8))
		rightSample := int16(uint16(pcmBytes[i*4+2]) | (uint16(pcmBytes[i*4+3]) << 8))

		// 通过平均值混合为单声道样本
		// 使用int32进行中间求和以防止在除法前溢出
		pcmMonoInt16[i] = int16((int32(leftSample) + int32(rightSample)) / 2)
	}

	// 将 []int16 类型的单声道PCM数据转换为 []byte (仍然是16位小端序)
	monoPcmDataBytes := make([]byte, numMonoSamples*2) // 每个int16样本占用2字节
	for i, sample := range pcmMonoInt16 {
		monoPcmDataBytes[i*2] = byte(sample)        // 低字节 (LSB)
		monoPcmDataBytes[i*2+1] = byte(sample >> 8) // 高字节 (MSB)
	}

	//音频播放时长
	duration := float64(numMonoSamples) / float64(mp3SampleRate) // 单声道PCM数据的时长 (秒)

	// 函数签名要求返回 [][]byte.
	// 我们将整个单声道PCM数据作为外部切片中的单个段/切片返回.
	return [][]byte{monoPcmDataBytes}, duration, nil
}

// CopyAudioFile 复制音频文件
func CopyAudioFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// SaveAudioFile 保存音频数据到文件
func SaveAudioFile(data []byte, filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("写入音频数据失败: %v", err)
	}

	return nil
}
