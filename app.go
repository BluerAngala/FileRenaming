package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx     context.Context
	apiKey  string // SiliconFlow API Key
	baseURL string // SiliconFlow API Base URL
	model   string // AI Model name
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// FileInfo 文件信息
type FileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	IsDir    bool   `json:"isDir"`
	FullPath string `json:"fullPath"`
}

// RenameRule 重命名规则
type RenameRule struct {
	Mode string `json:"mode"` // 模式: "rule" 或 "ai"

	// 规则模式字段
	Pattern     string `json:"pattern"`     // 匹配模式，如 "*.txt"
	ReplaceFrom string `json:"replaceFrom"` // 替换源
	ReplaceTo   string `json:"replaceTo"`   // 替换目标
	AddPrefix   string `json:"addPrefix"`   // 添加前缀
	AddSuffix   string `json:"addSuffix"`   // 添加后缀
	CaseType    string `json:"caseType"`    // 大小写类型: "lower", "upper", "title"
	NumberStart int    `json:"numberStart"` // 数字起始值
	NumberStep  int    `json:"numberStep"`  // 数字步长

	// AI模式字段
	AIPrompt    string   `json:"aiPrompt"`    // 用户输入的AI提示词
	AIGenerated []string `json:"aiGenerated"` // AI生成的新文件名列表（不含扩展名）
}

// OpenFileDialog 打开文件选择对话框
func (a *App) OpenFileDialog() ([]string, error) {
	selection, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return selection, nil
}

// OpenFolderDialog 打开文件夹选择对话框
func (a *App) OpenFolderDialog() ([]string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择文件夹",
	})
	if err != nil {
		return nil, err
	}
	if folder == "" {
		return nil, nil
	}
	return []string{folder}, nil
}

// GetFilesFromPaths 从路径列表获取文件信息
// recursive: true 表示递归获取文件夹内的所有文件，false 表示只处理直接拖放的文件
func (a *App) GetFilesFromPaths(paths []string, recursive bool) ([]FileInfo, error) {
	var files []FileInfo

	for _, path := range paths {
		// 清理路径
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			if recursive {
				// 如果是文件夹且需要递归，获取文件夹内的所有文件
				err := filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if !fileInfo.IsDir() {
						files = append(files, FileInfo{
							Path:     filepath.Dir(filePath),
							Name:     fileInfo.Name(),
							IsDir:    false,
							FullPath: filePath,
						})
					}
					return nil
				})
				if err != nil {
					continue
				}
			}
			// 如果不递归，跳过文件夹
		} else {
			// 直接是文件，添加到列表
			files = append(files, FileInfo{
				Path:     filepath.Dir(path),
				Name:     info.Name(),
				IsDir:    false,
				FullPath: path,
			})
		}
	}

	return files, nil
}

// GetFilesFromFileInfos 直接从 FileInfo 列表获取（用于前端已提供完整信息的场景）
func (a *App) GetFilesFromFileInfos(fileInfos []FileInfo) ([]FileInfo, error) {
	var validFiles []FileInfo

	for _, fileInfo := range fileInfos {
		// 验证文件是否存在
		if fileInfo.FullPath != "" {
			_, err := os.Stat(fileInfo.FullPath)
			if err == nil {
				validFiles = append(validFiles, fileInfo)
			}
		} else if fileInfo.Path != "" && fileInfo.Name != "" {
			// 尝试组合路径
			fullPath := filepath.Join(fileInfo.Path, fileInfo.Name)
			_, err := os.Stat(fullPath)
			if err == nil {
				fileInfo.FullPath = fullPath
				validFiles = append(validFiles, fileInfo)
			}
		}
	}

	return validFiles, nil
}

// RenameFiles 批量重命名文件
func (a *App) RenameFiles(files []FileInfo, rule RenameRule) ([]string, error) {
	var errors []string
	numberCounter := rule.NumberStart

	for _, file := range files {
		// 检查文件是否匹配模式
		if rule.Pattern != "" && rule.Pattern != "*" {
			matched, err := filepath.Match(rule.Pattern, file.Name)
			if err != nil || !matched {
				continue
			}
		}

		// 获取文件名和扩展名
		ext := filepath.Ext(file.Name)
		nameWithoutExt := strings.TrimSuffix(file.Name, ext)

		// 应用替换规则
		newName := nameWithoutExt
		if rule.ReplaceFrom != "" {
			newName = strings.ReplaceAll(newName, rule.ReplaceFrom, rule.ReplaceTo)
		}

		// 应用大小写规则
		switch rule.CaseType {
		case "lower":
			newName = strings.ToLower(newName)
		case "upper":
			newName = strings.ToUpper(newName)
		case "title":
			lowerName := strings.ToLower(newName)
			if len(lowerName) > 0 {
				newName = strings.ToUpper(string(lowerName[0])) + lowerName[1:]
			}
		}

		// 添加前缀和后缀
		if rule.AddPrefix != "" {
			newName = rule.AddPrefix + newName
		}
		if rule.AddSuffix != "" {
			newName = newName + rule.AddSuffix
		}

		// 添加数字编号
		if rule.NumberStart > 0 || rule.NumberStep > 0 {
			newName = fmt.Sprintf("%s_%d", newName, numberCounter)
			numberCounter += rule.NumberStep
			if rule.NumberStep == 0 {
				numberCounter++
			}
		}

		// 组合新文件名
		newFileName := newName + ext
		newPath := filepath.Join(file.Path, newFileName)

		// 如果新文件名和原文件名相同，跳过
		if newPath == file.FullPath {
			continue
		}

		// 如果目标文件已存在，添加序号
		counter := 1
		for {
			_, err := os.Stat(newPath)
			if os.IsNotExist(err) {
				break
			}
			newNameWithCounter := fmt.Sprintf("%s_%d", newName, counter)
			newPath = filepath.Join(file.Path, newNameWithCounter+ext)
			counter++
			if counter > 1000 { // 防止无限循环
				errors = append(errors, fmt.Sprintf("无法重命名 %s: 目标文件已存在", file.Name))
				break
			}
		}

		// 执行重命名
		if counter <= 1000 {
			err := os.Rename(file.FullPath, newPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("重命名 %s 失败: %v", file.Name, err))
			}
		}
	}

	if len(errors) > 0 {
		return errors, fmt.Errorf("部分文件重命名失败")
	}

	return nil, nil
}

// SetAIAPIKey 设置AI API密钥和BaseURL
func (a *App) SetAIAPIKey(apiKey string, baseURL string) error {
	if apiKey == "" {
		return fmt.Errorf("API密钥不能为空")
	}
	a.apiKey = apiKey
	if baseURL == "" {
		// 默认使用SiliconFlow的API地址
		a.baseURL = "https://api.siliconflow.cn/v1"
	} else {
		a.baseURL = baseURL
	}
	// 默认使用SiliconFlow支持的模型
	if a.model == "" {
		a.model = "deepseek-ai/DeepSeek-V3" // SiliconFlow支持的模型
	}
	return nil
}

// SetAIModel 设置AI模型名称
func (a *App) SetAIModel(model string) error {
	if model == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	a.model = model
	return nil
}

// AIConfig AI配置结构
type AIConfig struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
	Model   string `json:"model"`
}

// getConfigPath 获取配置文件路径
func (a *App) getConfigPath() (string, error) {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("获取配置目录失败: %v", err)
	}

	// 创建应用配置目录
	appConfigDir := filepath.Join(configDir, "FileRenaming")
	err = os.MkdirAll(appConfigDir, 0755)
	if err != nil {
		return "", fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 配置文件路径
	configPath := filepath.Join(appConfigDir, "ai_config.json")
	return configPath, nil
}

// SaveAIConfig 保存AI配置到文件
func (a *App) SaveAIConfig(config AIConfig) error {
	configPath, err := a.getConfigPath()
	if err != nil {
		return err
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// LoadAIConfig 从文件加载AI配置
func (a *App) LoadAIConfig() (AIConfig, error) {
	var config AIConfig

	configPath, err := a.getConfigPath()
	if err != nil {
		return config, err
	}

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 文件不存在，返回空配置
		return config, nil
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析JSON
	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return config, nil
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse 模型列表响应
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// GetAIModels 获取可用的AI模型列表
func (a *App) GetAIModels(modelType string) ([]ModelInfo, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("请先设置AI API密钥")
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/models", a.baseURL)
	if modelType != "" {
		url += fmt.Sprintf("?type=%s", modelType)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("API错误: %s (code: %d)", errorResp.Message, errorResp.Code)
		}
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	return modelsResp.Data, nil
}

// GenerateAINames 使用AI生成新文件名
func (a *App) GenerateAINames(files []FileInfo, prompt string) ([]string, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("请先设置AI API密钥")
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("文件列表为空")
	}

	// 构建文件列表上下文
	fileList := make([]string, 0, len(files))
	for _, file := range files {
		ext := filepath.Ext(file.Name)
		nameWithoutExt := strings.TrimSuffix(file.Name, ext)
		fileList = append(fileList, fmt.Sprintf("原文件名: %s (扩展名: %s)", nameWithoutExt, ext))
	}

	// 构建AI提示词
	systemPrompt := `你是一个文件重命名助手。用户会提供一批文件的原始文件名，你需要根据用户的需求为每个文件生成一个新的文件名（不含扩展名）。

要求：
1. 只返回新文件名，不要包含扩展名
2. 文件名应该简洁、有意义、符合用户需求
3. 返回JSON数组格式，数组中的每个元素对应一个文件的新文件名
4. 数组顺序必须与输入的文件顺序完全一致
5. 只返回JSON数组，不要有其他说明文字

示例输入：
原文件名: 测试文件1 (扩展名: .txt)
原文件名: 测试文件2 (扩展名: .jpg)

示例输出（如果用户要求添加"新"前缀）：
["新测试文件1", "新测试文件2"]`

	userPrompt := fmt.Sprintf(`文件列表：
%s

用户需求：%s

请根据用户需求为每个文件生成新文件名，返回JSON数组格式。`, strings.Join(fileList, "\n"), prompt)

	// 创建OpenAI客户端（使用SiliconFlow）
	config := openai.DefaultConfig(a.apiKey)
	config.BaseURL = a.baseURL
	client := openai.NewClientWithConfig(config)

	// 确定使用的模型
	modelName := a.model
	if modelName == "" {
		// 如果未设置模型，使用SiliconFlow支持的默认模型
		modelName = "deepseek-ai/DeepSeek-V3"
	}

	// 调用AI API
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.7,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("AI API调用失败: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("AI未返回结果")
	}

	// 解析AI返回的JSON数组
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	// 移除可能的markdown代码块标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var newNames []string
	if err := json.Unmarshal([]byte(content), &newNames); err != nil {
		return nil, fmt.Errorf("解析AI返回结果失败: %v, 原始内容: %s", err, content)
	}

	// 验证返回的文件名数量是否匹配
	if len(newNames) != len(files) {
		return nil, fmt.Errorf("AI返回的文件名数量(%d)与输入文件数量(%d)不匹配", len(newNames), len(files))
	}

	return newNames, nil
}

// AIRenameFiles 使用AI生成的文件名进行重命名
func (a *App) AIRenameFiles(files []FileInfo, newNames []string) ([]string, error) {
	if len(files) != len(newNames) {
		return nil, fmt.Errorf("文件数量(%d)与新文件名数量(%d)不匹配", len(files), len(newNames))
	}

	var errors []string

	for i, file := range files {
		// 获取文件扩展名
		ext := filepath.Ext(file.Name)

		// 组合新文件名（AI返回的已经不含扩展名）
		newFileName := newNames[i] + ext
		newPath := filepath.Join(file.Path, newFileName)

		// 如果新文件名和原文件名相同，跳过
		if newPath == file.FullPath {
			continue
		}

		// 如果目标文件已存在，添加序号
		counter := 1
		for {
			_, err := os.Stat(newPath)
			if os.IsNotExist(err) {
				break
			}
			newNameWithCounter := fmt.Sprintf("%s_%d", newNames[i], counter)
			newPath = filepath.Join(file.Path, newNameWithCounter+ext)
			counter++
			if counter > 1000 { // 防止无限循环
				errors = append(errors, fmt.Sprintf("无法重命名 %s: 目标文件已存在", file.Name))
				break
			}
		}

		// 执行重命名
		if counter <= 1000 {
			err := os.Rename(file.FullPath, newPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("重命名 %s 失败: %v", file.Name, err))
			}
		}
	}

	if len(errors) > 0 {
		return errors, fmt.Errorf("部分文件重命名失败")
	}

	return nil, nil
}

// PromptTemplate 提示词模板结构
type PromptTemplate struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// PromptTemplates 提示词模板列表
type PromptTemplates struct {
	Templates []PromptTemplate `json:"templates"`
}

// getPromptTemplatesPath 获取提示词模板文件路径
func (a *App) getPromptTemplatesPath() (string, error) {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("获取配置目录失败: %v", err)
	}

	// 创建应用配置目录
	appConfigDir := filepath.Join(configDir, "FileRenaming")
	err = os.MkdirAll(appConfigDir, 0755)
	if err != nil {
		return "", fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 模板文件路径
	templatesPath := filepath.Join(appConfigDir, "prompt_templates.json")
	return templatesPath, nil
}

// SavePromptTemplates 保存提示词模板列表
func (a *App) SavePromptTemplates(templates PromptTemplates) error {
	templatesPath, err := a.getPromptTemplatesPath()
	if err != nil {
		return err
	}

	// 序列化模板
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化模板失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(templatesPath, data, 0644)
	if err != nil {
		return fmt.Errorf("写入模板文件失败: %v", err)
	}

	return nil
}

// LoadPromptTemplates 加载提示词模板列表
func (a *App) LoadPromptTemplates() (PromptTemplates, error) {
	var templates PromptTemplates

	templatesPath, err := a.getPromptTemplatesPath()
	if err != nil {
		return templates, err
	}

	// 检查文件是否存在
	if _, err := os.Stat(templatesPath); os.IsNotExist(err) {
		// 文件不存在，返回默认模板
		return a.getDefaultTemplates(), nil
	}

	// 读取文件
	data, err := os.ReadFile(templatesPath)
	if err != nil {
		return templates, fmt.Errorf("读取模板文件失败: %v", err)
	}

	// 解析JSON
	err = json.Unmarshal(data, &templates)
	if err != nil {
		return templates, fmt.Errorf("解析模板文件失败: %v", err)
	}

	return templates, nil
}

// getDefaultTemplates 获取默认提示词模板
func (a *App) getDefaultTemplates() PromptTemplates {
	return PromptTemplates{
		Templates: []PromptTemplate{
			{
				Name:    "日期+文件名+序号",
				Content: "按照今天日期+源文件名称+序号命名，格式：YYYYMMDD_原文件名_序号",
			},
			{
				Name:    "文件名+序号",
				Content: "在文件名后添加序号，格式：原文件名_序号",
			},
			{
				Name:    "日期+序号",
				Content: "使用今天日期和序号命名，格式：YYYYMMDD_序号",
			},
			{
				Name:    "序号+文件名",
				Content: "在文件名前添加序号，格式：序号_原文件名",
			},
			{
				Name:    "清理文件名",
				Content: "清理文件名中的特殊字符和多余空格，保留中文、英文、数字和基本标点",
			},
			{
				Name:    "统一小写",
				Content: "将文件名转换为小写，并保持文件扩展名不变",
			},
			{
				Name:    "统一大写",
				Content: "将文件名转换为大写，并保持文件扩展名不变",
			},
			{
				Name:    "首字母大写",
				Content: "将文件名首字母大写，其余小写，并保持文件扩展名不变",
			},
		},
	}
}
