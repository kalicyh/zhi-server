package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"xiaozhi-server-go/src/configs"
	"xiaozhi-server-go/src/core/chat"
	"xiaozhi-server-go/src/core/providers"
	"xiaozhi-server-go/src/core/utils"
)

// ConnectionHandler 连接处理器结构
type ConnectionHandler struct {
	config    *configs.Config
	logger    *utils.Logger
	conn      Conn
	providers struct {
		asr providers.ASRProvider
		llm providers.LLMProvider
		tts providers.TTSProvider
	}

	// 会话相关
	sessionID    string
	headers      map[string]string
	clientIP     string
	clientIPInfo map[string]interface{}

	// 状态标志
	clientAbort      bool
	clientListenMode string
	isDeviceVerified bool
	closeAfterChat   bool

	// 语音处理相关
	clientAudioBuffer []byte
	clientHaveVoice   bool
	clientVoiceStop   bool
	voiceTimestamps   struct {
		haveVoiceLast time.Time
		noVoiceLast   time.Time
	}
	asrTicker          *time.Ticker // 用于定时检查ASR结果
	asr_server_receive bool         // 是否接收ASR结果

	// 对话相关
	dialogueManager      *chat.DialogueManager
	prompt               string
	welcomeMsg           map[string]interface{}
	tts_first_text_index int
	tts_last_text_index  int

	// 并发控制
	mu               sync.Mutex
	stopChan         chan struct{}
	clientAudioQueue chan []byte
	clientTextQueue  chan string

	// TTS任务队列
	ttsQueue chan struct {
		text      string
		textIndex int
	}

	audioMessagesQueue chan struct {
		filepath  string
		text      string
		textIndex int
	}
}

// NewConnectionHandler 创建新的连接处理器
func NewConnectionHandler(
	config *configs.Config,
	providers struct {
		asr providers.ASRProvider
		llm providers.LLMProvider
		tts providers.TTSProvider
	},
	logger *utils.Logger,
) *ConnectionHandler {
	handler := &ConnectionHandler{
		config:           config,
		providers:        providers,
		logger:           logger,
		clientListenMode: "auto",
		stopChan:         make(chan struct{}),
		clientAudioQueue: make(chan []byte, 100),
		clientTextQueue:  make(chan string, 100),
		ttsQueue: make(chan struct {
			text      string
			textIndex int
		}, 100),
		audioMessagesQueue: make(chan struct {
			filepath  string
			text      string
			textIndex int
		}, 100),

		asrTicker:            time.NewTicker(100 * time.Millisecond), // 每100ms检查一次ASR结果
		tts_last_text_index:  -1,
		tts_first_text_index: -1,
	}

	// 初始化对话管理器
	handler.dialogueManager = chat.NewDialogueManager(handler.logger, nil)

	return handler
}

// Handle 处理WebSocket连接
func (h *ConnectionHandler) Handle(conn Conn) {
	defer conn.Close()

	h.conn = conn

	// 发送欢迎消息
	if err := h.sendWelcomeMessage(); err != nil {
		h.logger.Error(fmt.Sprintf("发送欢迎消息失败: %v", err))
		return
	}

	// 启动消息处理协程
	go h.processClientAudioMessagesCoroutine() // 添加客户端音频消息处理协程
	go h.processClientTextMessagesCoroutine()  // 添加客户端文本消息处理协程
	go h.monitorASRResultsCoroutine()          // 添加监控ASR结果的协程
	go h.processTTSQueueCoroutine()            // 添加TTS队列处理协程
	go h.sendAudioMessageCoroutine()           // 添加音频消息发送协程

	// 主消息循环
	for {
		select {
		case <-h.stopChan:
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				h.logger.Error(fmt.Sprintf("读取消息失败: %v", err))
				return
			}

			if err := h.handleMessage(messageType, message); err != nil {
				h.logger.Error(fmt.Sprintf("处理消息失败: %v", err))
				if h.closeAfterChat {
					return
				}
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (h *ConnectionHandler) handleMessage(messageType int, message []byte) error {
	switch messageType {
	case 1: // 文本消息
		h.clientTextQueue <- string(message)
		return nil
	case 2: // 二进制消息（音频数据）
		h.clientAudioQueue <- message
		return nil
	default:
		h.logger.Error(fmt.Sprintf("未知的消息类型: %d", messageType))
		return fmt.Errorf("未知的消息类型: %d", messageType)
	}
}

// processClientTextMessagesCoroutine 处理文本消息队列
func (h *ConnectionHandler) processClientTextMessagesCoroutine() {
	for {
		select {
		case <-h.stopChan:
			return
		case text := <-h.clientTextQueue:
			if err := h.processClientTextMessage(context.Background(), text); err != nil {
				h.logger.Error(fmt.Sprintf("处理文本数据失败: %v", err))
			}
		}
	}
}

// processClientAudioMessagesCoroutine 处理音频消息队列
func (h *ConnectionHandler) processClientAudioMessagesCoroutine() {
	for {
		select {
		case <-h.stopChan:
			return
		case audioData := <-h.clientAudioQueue:
			if err := h.providers.asr.AddAudio(audioData); err != nil {
				h.logger.Error(fmt.Sprintf("处理音频数据失败: %v", err))
			}
		}
	}
}

// 新增监控ASR结果的方法
func (h *ConnectionHandler) monitorASRResultsCoroutine() {
	for {
		select {
		case <-h.stopChan:
			return
		case <-h.asrTicker.C:
			// 定期获取ASR结果
			text, err := h.providers.asr.GetFinalResult()
			if err != nil {
				errMsg := err.Error()
				if errMsg == "未初始化流式识别" || errMsg == "ASR识别结果为空" {
					continue
				}
				if strings.Contains(errMsg, "读取最终响应失败") {
					continue
				}

				h.logger.Error(fmt.Sprintf("获取ASR结果失败: %v", err))
				h.providers.asr.Reset()
				continue
			}

			if text != "" {
				// 将文本放入队列进行处理
				h.logger.Info("ASR识别结果: " + text)
				h.asr_server_receive = false
				h.handleChatMessage(context.Background(), text)
			} else {
				h.logger.Info("ASR识别结果为空")
			}
		}
	}
}

// processClientTextMessage 处理文本数据
func (h *ConnectionHandler) processClientTextMessage(ctx context.Context, text string) error {
	// 解析JSON消息
	var msgJSON interface{}
	if err := json.Unmarshal([]byte(text), &msgJSON); err != nil {
		return h.conn.WriteMessage(1, []byte(text))
	}

	// 检查是否为整数类型
	if _, ok := msgJSON.(float64); ok {
		return h.conn.WriteMessage(1, []byte(text))
	}

	// 解析为map类型处理具体消息
	msgMap, ok := msgJSON.(map[string]interface{})
	if !ok {
		return fmt.Errorf("消息格式错误")
	}

	// 根据消息类型分发处理
	msgType, ok := msgMap["type"].(string)
	if !ok {
		return fmt.Errorf("消息类型错误")
	}

	switch msgType {
	case "hello":
		return h.handleHelloMessage(msgMap)
	case "abort":
		return h.handleAbortMessage()
	case "listen":
		return h.handleListenMessage(msgMap)
	case "iot":
		return h.handleIotMessage(msgMap)
	case "chat":
		return h.handleChatMessage(ctx, text)
	default:
		return fmt.Errorf("未知的消息类型: %s", msgType)
	}
}

// handleHelloMessage 处理欢迎消息
// 客户端会上传语音格式和采样率等信息
// 这里可以根据需要进行处理
func (h *ConnectionHandler) handleHelloMessage(msgMap map[string]interface{}) error {
	h.logger.Info("收到客户端欢迎消息: " + fmt.Sprintf("%v", msgMap))
	return nil
}

// handleAbortMessage 处理中止消息
func (h *ConnectionHandler) handleAbortMessage() error {
	h.clientAbort = true
	return nil
}

// handleListenMessage 处理语音相关消息
func (h *ConnectionHandler) handleListenMessage(msgMap map[string]interface{}) error {
	// 处理mode参数
	if mode, ok := msgMap["mode"].(string); ok {
		h.clientListenMode = mode
		h.logger.Info(fmt.Sprintf("客户端拾音模式：%s", h.clientListenMode))
	}

	// 处理state参数
	state, ok := msgMap["state"].(string)
	if !ok {
		return fmt.Errorf("listen消息缺少state参数")
	}

	switch state {
	case "start":
		h.clientHaveVoice = true
		h.clientVoiceStop = false
	case "stop":
		h.clientHaveVoice = true
		h.clientVoiceStop = true
		if len(h.clientAudioBuffer) > 0 {
			h.clientAudioQueue <- []byte{}
		}
	case "detect":
		h.clientHaveVoice = false
		h.clientAudioBuffer = nil
		// 处理text参数
		if text, ok := msgMap["text"].(string); ok {
			// TODO: 实现去除标点和长度的函数
			// _, text = removePunctuationAndLength(text)
			return h.handleChatMessage(context.Background(), text)
		}
	}
	return nil
}

// handleIotMessage 处理IOT设备消息
func (h *ConnectionHandler) handleIotMessage(msgMap map[string]interface{}) error {
	if descriptors, ok := msgMap["descriptors"].([]interface{}); ok {
		// 处理设备描述符
		// 这里需要实现具体的IOT设备描述符处理逻辑
		h.logger.Info(fmt.Sprintf("收到IOT设备描述符：%v", descriptors))
	}
	if states, ok := msgMap["states"].([]interface{}); ok {
		// 处理设备状态
		// 这里需要实现具体的IOT设备状态处理逻辑
		h.logger.Info(fmt.Sprintf("收到IOT设备状态：%v", states))
	}
	return nil
}

// sendEmotionMessage 发送情绪消息
func (h *ConnectionHandler) sendEmotionMessage(emotion string) error {
	data := map[string]interface{}{
		"type":       "llm",
		"text":       utils.GetEmotionEmoji(emotion),
		"emotion":    emotion,
		"session_id": h.sessionID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化情绪消息失败: %v", err)
	}
	return h.conn.WriteMessage(1, jsonData)
}

// handleChatMessage 处理聊天消息
func (h *ConnectionHandler) handleChatMessage(ctx context.Context, text string) error {
	// 判断是否需要验证
	if h.isNeedAuth() {
		if err := h.checkAndBroadcastAuthCode(); err != nil {
			return err
		}
		return nil
	}

	// 立即发送 stt 消息
	err := h.sendSTTMessage(text)
	if err != nil {
		return fmt.Errorf("发送STT消息失败: %v", err)
	}

	// 发送tts start状态
	if err := h.sendTTSMessage("start", "", 0); err != nil {
		return fmt.Errorf("发送TTS开始状态失败: %v", err)
	}

	// 发送思考状态的情绪
	if err := h.sendEmotionMessage("thinking"); err != nil {
		return fmt.Errorf("发送情绪消息失败: %v", err)
	}

	h.logger.Info("收到聊天消息: " + text)

	// 添加用户消息到对话历史
	h.dialogueManager.Put(chat.Message{
		Role:    "user",
		Content: text,
	})

	// 转换消息格式并使用LLM生成回复
	messages := make([]providers.Message, 0)
	for _, msg := range h.dialogueManager.GetLLMDialogue() {
		messages = append(messages, providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 使用LLM生成回复
	responses, err := h.providers.llm.Response(ctx, h.sessionID, messages)
	if err != nil {
		return fmt.Errorf("LLM生成回复失败: %v", err)
	}

	// 处理回复
	var responseMessage []string
	processedChars := 0
	textIndex := 0

	for content := range responses {
		responseMessage = append(responseMessage, content)
		if h.clientAbort {
			break
		}

		// 处理分段
		fullText := joinStrings(responseMessage)
		currentText := fullText[processedChars:]

		//h.logger.Info("LLM生成回复: " + fmt.Sprintf("%s", fullText))

		// 按标点符号分割
		if segment, chars := splitAtLastPunctuation(currentText); chars > 0 {
			textIndex++
			h.recode_first_last_text(segment, textIndex)
			if err := h.speakAndPlay(segment, textIndex); err != nil {
				h.logger.Error(fmt.Sprintf("语音合成失败:%s %v", segment, err))
			} else {
				h.logger.Info("语音合成成功: " + segment)
			}
			processedChars += chars
		}
	}

	// 处理剩余文本
	remainingText := joinStrings(responseMessage)[processedChars:]
	if remainingText != "" {
		textIndex++
		h.recode_first_last_text(remainingText, textIndex)
		if err := h.speakAndPlay(remainingText, textIndex); err != nil {
			h.logger.Error(fmt.Sprintf("语音合成失败: %v", err))
		} else {
			h.logger.Info("语音合成成功: " + remainingText)
		}
	}

	// 分析回复并发送相应的情绪
	content := joinStrings(responseMessage)

	// 添加助手回复到对话历史
	h.dialogueManager.Put(chat.Message{
		Role:    "assistant",
		Content: content,
	})

	return nil
}

// isNeedAuth 判断是否需要验证
func (h *ConnectionHandler) isNeedAuth() bool {
	if !h.config.Server.Auth.Enabled {
		return false
	}
	return !h.isDeviceVerified
}

// checkAndBroadcastAuthCode 检查并广播认证码
func (h *ConnectionHandler) checkAndBroadcastAuthCode() error {
	// 这里简化了认证逻辑，实际需要根据具体需求实现
	text := "请联系管理员进行设备认证"
	return h.speakAndPlay(text, 0)
}

// processTTSQueueCoroutine 处理TTS队列
func (h *ConnectionHandler) processTTSQueueCoroutine() {
	for {
		select {
		case <-h.stopChan:
			return
		case task := <-h.ttsQueue:
			h.processTTSTask(task.text, task.textIndex)
		}
	}
}

func (h *ConnectionHandler) sendAudioMessageCoroutine() {
	for {
		select {
		case <-h.stopChan:
			return
		case task := <-h.audioMessagesQueue:
			h.sendAudioMessage(task.filepath, task.text, task.textIndex)
		}
	}
}

func (h *ConnectionHandler) sendAudioMessage(filepath string, text string, textIndex int) {
	if len(filepath) == 0 {
		return
	}

	defer func() {
		if textIndex == h.tts_last_text_index {
			h.sendTTSMessage("stop", "", textIndex)
			h.clearSpeakStatus()
		}
	}()

	audioData, duration, err := utils.AudioToPCMData(filepath)
	if err != nil {
		h.logger.Error(fmt.Sprintf("音频转换失败: %v", err))
		return
	}

	fmt.Println("音频时长:", duration)

	// 发送TTS状态开始通知
	if err := h.sendTTSMessage("sentence_start", text, textIndex); err != nil {
		h.logger.Error(fmt.Sprintf("发送TTS开始状态失败: %v", err))
		return
	}

	// 发送音频数据
	for _, chunk := range audioData {
		if err := h.conn.WriteMessage(2, chunk); err != nil {
			h.logger.Error(fmt.Sprintf("发送音频数据失败: %v", err))
			return
		}
	}
	h.logger.Info(fmt.Sprintf("TTS发送: \"%s\" (索引:%d)", text, textIndex))
	now := time.Now()
	time.Sleep(time.Duration(duration*1000) * time.Millisecond)
	spent := time.Since(now)
	h.logger.Info(fmt.Sprintf("音频数据发送完成, 休眠: %v", spent))
	// 发送TTS状态结束通知
	if err := h.sendTTSMessage("sentence_end", text, textIndex); err != nil {
		h.logger.Error(fmt.Sprintf("发送TTS结束状态失败: %v", err))
		return
	}
}

// processTTSTask 处理单个TTS任务
func (h *ConnectionHandler) processTTSTask(text string, textIndex int) {
	if text == "" {
		return
	}

	// 生成语音文件
	filepath, err := h.providers.tts.ToTTS(text)
	if err != nil {
		h.logger.Error(fmt.Sprintf("TTS转换失败: %v", err))
		return
	}

	h.audioMessagesQueue <- struct {
		filepath  string
		text      string
		textIndex int
	}{filepath, text, textIndex}
}

// speakAndPlay 合成并播放语音
func (h *ConnectionHandler) speakAndPlay(text string, textIndex int) error {
	if text == "" {
		return nil
	}
	// 将任务加入队列，不阻塞当前流程
	h.ttsQueue <- struct {
		text      string
		textIndex int
	}{text, textIndex}

	return nil
}

func (h *ConnectionHandler) sendTTSMessage(state string, text string, textIndex int) error {
	// 发送TTS状态结束通知
	stateMsg := map[string]interface{}{
		"type":       "tts",
		"state":      state,
		"session_id": h.sessionID,
		"text":       text,
		"index":      textIndex,
	}
	data, err := json.Marshal(stateMsg)
	if err != nil {
		return fmt.Errorf("序列化%s状态失败: %v", state, err)
	}
	if err := h.conn.WriteMessage(1, data); err != nil {
		return fmt.Errorf("发送%s状态失败: %v", state, err)
	}
	return nil
}

func (h *ConnectionHandler) sendSTTMessage(text string) error {

	// 立即发送 stt 消息
	sttMsg := map[string]interface{}{
		"type":       "stt",
		"text":       text,
		"session_id": h.sessionID,
	}
	jsonData, err := json.Marshal(sttMsg)
	if err != nil {
		return fmt.Errorf("序列化 STT 消息失败: %v", err)
	}
	if err := h.conn.WriteMessage(1, jsonData); err != nil {
		return fmt.Errorf("发送 STT 消息失败: %v", err)
	}

	return nil
}

func (h *ConnectionHandler) clearSpeakStatus() {
	h.logger.Info("清除服务端讲话状态 ")
	h.asr_server_receive = true
	h.tts_last_text_index = -1
	h.tts_first_text_index = -1
	h.providers.asr.Reset() // 重置ASR状态
}

func (h *ConnectionHandler) recode_first_last_text(text string, text_index int) {
	if h.tts_first_text_index == -1 {
		h.logger.Info("大模型说出第一句话", text)
		h.tts_first_text_index = text_index
	}

	h.tts_last_text_index = text_index
}

// joinStrings 连接字符串切片
func joinStrings(strs []string) string {
	var result string
	for _, s := range strs {
		result += s
	}
	return result
}

// splitAtLastPunctuation 在最后一个标点符号处分割文本
func splitAtLastPunctuation(text string) (string, int) {
	punctuations := []string{"。", "？", "！", "；", "："}
	lastIndex := -1

	for _, punct := range punctuations {
		if idx := strings.LastIndex(text, punct); idx > lastIndex {
			lastIndex = idx
		}
	}

	if lastIndex == -1 {
		return "", 0
	}

	return text[:lastIndex+len("。")], lastIndex + len("。")
}

// sendWelcomeMessage 发送欢迎消息
func (h *ConnectionHandler) sendWelcomeMessage() error {
	welcome := h.config.Xiaozhi
	welcome["session_id"] = h.sessionID

	data, err := json.Marshal(welcome)
	if err != nil {
		return fmt.Errorf("序列化欢迎消息失败: %v", err)
	}

	return h.conn.WriteMessage(1, data)
}

// Close 清理资源
func (h *ConnectionHandler) Close() {
	h.asrTicker.Stop()
	close(h.stopChan)
	close(h.clientAudioQueue)
	close(h.clientTextQueue)
}
