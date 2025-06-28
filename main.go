package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	MCPVersion    = "2024-11-05"
	ServerName    = "criage-mcp-server"
	ServerVersion = "1.0.0"
)

// MCP Protocol structures
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	server := NewMCPServer()
	server.Run()
}

type MCPServer struct {
	packageManager *PackageManager
}

func NewMCPServer() *MCPServer {
	pm, err := NewPackageManager()
	if err != nil {
		log.Fatalf("Не удалось создать пакетный менеджер: %v", err)
	}

	return &MCPServer{
		packageManager: pm,
	}
}

func (s *MCPServer) Run() {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var message MCPMessage
		if err := decoder.Decode(&message); err != nil {
			log.Printf("Ошибка декодирования сообщения: %v", err)
			continue
		}

		response := s.handleMessage(message)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				log.Printf("Ошибка кодирования ответа: %v", err)
			}
		}
	}
}

func (s *MCPServer) handleMessage(message MCPMessage) *MCPMessage {
	switch message.Method {
	case "initialize":
		return s.handleInitialize(message)
	case "tools/list":
		return s.handleToolsList(message)
	case "tools/call":
		return s.handleToolsCall(message)
	default:
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Неизвестный метод: %s", message.Method),
			},
		}
	}
}

func (s *MCPServer) handleInitialize(message MCPMessage) *MCPMessage {
	result := InitializeResult{
		ProtocolVersion: MCPVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		ServerInfo: ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}

	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  result,
	}
}

func (s *MCPServer) handleToolsList(message MCPMessage) *MCPMessage {
	tools := []Tool{
		{
			Name:        "install_package",
			Description: "Устанавливает пакет из репозитория Criage",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Имя пакета для установки",
					},
					"version": map[string]interface{}{
						"type":        "string",
						"description": "Версия пакета (необязательно)",
					},
					"global": map[string]interface{}{
						"type":        "boolean",
						"description": "Глобальная установка",
						"default":     false,
					},
					"force": map[string]interface{}{
						"type":        "boolean",
						"description": "Принудительная переустановка",
						"default":     false,
					},
					"arch": map[string]interface{}{
						"type":        "string",
						"description": "Целевая архитектура",
					},
					"os": map[string]interface{}{
						"type":        "string",
						"description": "Целевая операционная система",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "uninstall_package",
			Description: "Удаляет установленный пакет",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Имя пакета для удаления",
					},
					"global": map[string]interface{}{
						"type":        "boolean",
						"description": "Глобальное удаление",
						"default":     false,
					},
					"purge": map[string]interface{}{
						"type":        "boolean",
						"description": "Полное удаление с конфигурацией",
						"default":     false,
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "search_packages",
			Description: "Поиск пакетов в репозитории",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Поисковый запрос",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_packages",
			Description: "Показывает список установленных пакетов",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"global": map[string]interface{}{
						"type":        "boolean",
						"description": "Показать глобальные пакеты",
						"default":     false,
					},
					"outdated": map[string]interface{}{
						"type":        "boolean",
						"description": "Показать только устаревшие пакеты",
						"default":     false,
					},
				},
			},
		},
		{
			Name:        "package_info",
			Description: "Показывает информацию о пакете",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Имя пакета",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "update_package",
			Description: "Обновляет пакет до последней версии",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Имя пакета для обновления",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "create_package",
			Description: "Создает новый пакет",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Имя нового пакета",
					},
					"template": map[string]interface{}{
						"type":        "string",
						"description": "Шаблон пакета",
						"default":     "basic",
					},
					"author": map[string]interface{}{
						"type":        "string",
						"description": "Автор пакета",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Описание пакета",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "build_package",
			Description: "Собирает пакет",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"output_path": map[string]interface{}{
						"type":        "string",
						"description": "Путь для выходного файла",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Формат архива (criage, tar.zst, tar.gz и т.д.)",
						"default":     "criage",
					},
					"compression_level": map[string]interface{}{
						"type":        "integer",
						"description": "Уровень сжатия",
						"default":     3,
					},
				},
			},
		},
		{
			Name:        "publish_package",
			Description: "Публикует пакет в репозиторий",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"registry_url": map[string]interface{}{
						"type":        "string",
						"description": "URL репозитория",
					},
					"token": map[string]interface{}{
						"type":        "string",
						"description": "Токен аутентификации",
					},
				},
			},
		},
		{
			Name:        "repository_info",
			Description: "Получает информацию о репозитории",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL репозитория",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "refresh_repository_index",
			Description: "Принудительно обновляет индекс пакетов в репозитории (требует права администратора)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repository_url": map[string]interface{}{
						"type":        "string",
						"description": "URL репозитория для обновления индекса",
					},
					"auth_token": map[string]interface{}{
						"type":        "string",
						"description": "Токен авторизации для доступа к операциям администрирования",
					},
				},
				"required": []string{"repository_url", "auth_token"},
			},
		},
		{
			Name:        "get_repository_stats",
			Description: "Получает детальную статистику репозитория",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repository_url": map[string]interface{}{
						"type":        "string",
						"description": "URL репозитория для получения статистики",
					},
				},
				"required": []string{"repository_url"},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  result,
	}
}

func (s *MCPServer) handleToolsCall(message MCPMessage) *MCPMessage {
	var params CallToolParams
	paramBytes, _ := json.Marshal(message.Params)
	if err := json.Unmarshal(paramBytes, &params); err != nil {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Неверные параметры",
				Data:    err.Error(),
			},
		}
	}

	result, err := s.callTool(params.Name, params.Arguments)
	if err != nil {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      message.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Ошибка: %v", err),
				}},
				IsError: true,
			},
		}
	}

	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      message.ID,
		Result:  result,
	}
}

func (s *MCPServer) callTool(name string, args map[string]interface{}) (CallToolResult, error) {
	switch name {
	case "install_package":
		return s.installPackage(args)
	case "uninstall_package":
		return s.uninstallPackage(args)
	case "search_packages":
		return s.searchPackages(args)
	case "list_packages":
		return s.listPackages(args)
	case "package_info":
		return s.packageInfo(args)
	case "update_package":
		return s.updatePackage(args)
	case "create_package":
		return s.createPackage(args)
	case "build_package":
		return s.buildPackage(args)
	case "publish_package":
		return s.publishPackage(args)
	case "repository_info":
		return s.repositoryInfo(args)
	case "refresh_repository_index":
		return s.refreshRepositoryIndex(args)
	case "get_repository_stats":
		return s.getRepositoryStats(args)
	default:
		return CallToolResult{}, fmt.Errorf("неизвестный инструмент: %s", name)
	}
}

func getString(args map[string]interface{}, key string, defaultValue string) string {
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getBool(args map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getInt(args map[string]interface{}, key string, defaultValue int) int {
	if val, ok := args[key]; ok {
		if i, ok := val.(float64); ok {
			return int(i)
		}
	}
	return defaultValue
}

func (s *MCPServer) installPackage(args map[string]interface{}) (CallToolResult, error) {
	name := getString(args, "name", "")
	if name == "" {
		return CallToolResult{}, fmt.Errorf("имя пакета обязательно")
	}

	version := getString(args, "version", "")
	global := getBool(args, "global", false)
	force := getBool(args, "force", false)
	arch := getString(args, "arch", "")
	osName := getString(args, "os", "")

	err := s.packageManager.InstallPackage(name, version, global, force, false, arch, osName)
	if err != nil {
		return CallToolResult{}, err
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("Пакет %s успешно установлен", name),
		}},
	}, nil
}

func (s *MCPServer) uninstallPackage(args map[string]interface{}) (CallToolResult, error) {
	name := getString(args, "name", "")
	if name == "" {
		return CallToolResult{}, fmt.Errorf("имя пакета обязательно")
	}

	global := getBool(args, "global", false)
	purge := getBool(args, "purge", false)

	err := s.packageManager.UninstallPackage(name, global, purge)
	if err != nil {
		return CallToolResult{}, err
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("Пакет %s успешно удален", name),
		}},
	}, nil
}

func (s *MCPServer) searchPackages(args map[string]interface{}) (CallToolResult, error) {
	query := getString(args, "query", "")
	if query == "" {
		return CallToolResult{}, fmt.Errorf("поисковый запрос обязателен")
	}

	results, err := s.packageManager.SearchPackages(query)
	if err != nil {
		return CallToolResult{}, err
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Найдено пакетов: %d\n\n", len(results)))

	for _, result := range results {
		output.WriteString(fmt.Sprintf("📦 %s (%s)\n", result.Name, result.Version))
		output.WriteString(fmt.Sprintf("   Описание: %s\n", result.Description))
		output.WriteString(fmt.Sprintf("   Автор: %s\n", result.Author))
		output.WriteString(fmt.Sprintf("   Загрузок: %d\n\n", result.Downloads))
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output.String(),
		}},
	}, nil
}

func (s *MCPServer) listPackages(args map[string]interface{}) (CallToolResult, error) {
	global := getBool(args, "global", false)
	outdated := getBool(args, "outdated", false)

	packages, err := s.packageManager.ListPackages(global, outdated)
	if err != nil {
		return CallToolResult{}, err
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Установленных пакетов: %d\n\n", len(packages)))

	for _, pkg := range packages {
		output.WriteString(fmt.Sprintf("📦 %s (%s)\n", pkg.Name, pkg.Version))
		output.WriteString(fmt.Sprintf("   Путь: %s\n", pkg.InstallPath))
		output.WriteString(fmt.Sprintf("   Размер: %s\n", formatSize(pkg.Size)))
		output.WriteString(fmt.Sprintf("   Дата установки: %s\n\n", pkg.InstallDate.Format("2006-01-02 15:04:05")))
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output.String(),
		}},
	}, nil
}

func (s *MCPServer) packageInfo(args map[string]interface{}) (CallToolResult, error) {
	name := getString(args, "name", "")
	if name == "" {
		return CallToolResult{}, fmt.Errorf("имя пакета обязательно")
	}

	info, err := s.packageManager.GetPackageInfo(name)
	if err != nil {
		return CallToolResult{}, err
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📦 Информация о пакете: %s\n\n", info.Name))
	output.WriteString(fmt.Sprintf("Версия: %s\n", info.Version))
	output.WriteString(fmt.Sprintf("Описание: %s\n", info.Description))
	output.WriteString(fmt.Sprintf("Автор: %s\n", info.Author))
	output.WriteString(fmt.Sprintf("Размер: %s\n", formatSize(info.Size)))
	output.WriteString(fmt.Sprintf("Путь установки: %s\n", info.InstallPath))
	output.WriteString(fmt.Sprintf("Дата установки: %s\n", info.InstallDate.Format("2006-01-02 15:04:05")))

	if len(info.Dependencies) > 0 {
		output.WriteString("\nЗависимости:\n")
		for name, version := range info.Dependencies {
			output.WriteString(fmt.Sprintf("  - %s: %s\n", name, version))
		}
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output.String(),
		}},
	}, nil
}

func (s *MCPServer) updatePackage(args map[string]interface{}) (CallToolResult, error) {
	name := getString(args, "name", "")
	if name == "" {
		return CallToolResult{}, fmt.Errorf("имя пакета обязательно")
	}

	err := s.packageManager.UpdatePackage(name)
	if err != nil {
		return CallToolResult{}, err
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("Пакет %s успешно обновлен", name),
		}},
	}, nil
}

func (s *MCPServer) createPackage(args map[string]interface{}) (CallToolResult, error) {
	name := getString(args, "name", "")
	if name == "" {
		return CallToolResult{}, fmt.Errorf("имя пакета обязательно")
	}

	template := getString(args, "template", "basic")
	author := getString(args, "author", "")
	description := getString(args, "description", "")

	err := s.packageManager.CreatePackage(name, template, author, description)
	if err != nil {
		return CallToolResult{}, err
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("Пакет %s успешно создан", name),
		}},
	}, nil
}

func (s *MCPServer) buildPackage(args map[string]interface{}) (CallToolResult, error) {
	outputPath := getString(args, "output_path", "")
	format := getString(args, "format", "criage")
	compressionLevel := getInt(args, "compression_level", 3)

	err := s.packageManager.BuildPackage(outputPath, format, compressionLevel)
	if err != nil {
		return CallToolResult{}, err
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: "Пакет успешно собран",
		}},
	}, nil
}

func (s *MCPServer) publishPackage(args map[string]interface{}) (CallToolResult, error) {
	registryURL := getString(args, "registry_url", "")
	token := getString(args, "token", "")

	err := s.packageManager.PublishPackage(registryURL, token)
	if err != nil {
		return CallToolResult{}, err
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: "Пакет успешно опубликован",
		}},
	}, nil
}

func (s *MCPServer) repositoryInfo(args map[string]interface{}) (CallToolResult, error) {
	url := getString(args, "url", "")
	if url == "" {
		return CallToolResult{}, fmt.Errorf("URL репозитория обязателен")
	}

	// Здесь будет логика получения информации о репозитории
	// Пока возвращаем заглушку
	var output strings.Builder
	output.WriteString(fmt.Sprintf("📊 Информация о репозитории: %s\n\n", url))
	output.WriteString("Состояние: Доступен\n")
	output.WriteString("API версия: v1\n")

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output.String(),
		}},
	}, nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// refreshRepositoryIndex принудительно обновляет индекс пакетов в репозитории
func (s *MCPServer) refreshRepositoryIndex(args map[string]interface{}) (CallToolResult, error) {
	repositoryURL := getString(args, "repository_url", "")
	if repositoryURL == "" {
		return CallToolResult{}, fmt.Errorf("URL репозитория обязателен")
	}

	authToken := getString(args, "auth_token", "")
	if authToken == "" {
		return CallToolResult{}, fmt.Errorf("токен авторизации обязателен")
	}

	err := s.packageManager.RefreshRepositoryIndex(repositoryURL, authToken)
	if err != nil {
		return CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: fmt.Sprintf("❌ Ошибка обновления индекса репозитория: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("✅ Индекс репозитория %s успешно обновлен", repositoryURL),
		}},
	}, nil
}

// getRepositoryStats получает детальную статистику репозитория
func (s *MCPServer) getRepositoryStats(args map[string]interface{}) (CallToolResult, error) {
	repositoryURL := getString(args, "repository_url", "")
	if repositoryURL == "" {
		return CallToolResult{}, fmt.Errorf("URL репозитория обязателен")
	}

	stats, err := s.packageManager.GetRepositoryStats(repositoryURL)
	if err != nil {
		return CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: fmt.Sprintf("❌ Ошибка получения статистики репозитория: %v", err),
			}},
			IsError: true,
		}, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("📊 Статистика репозитория: %s\n\n", repositoryURL))
	output.WriteString(fmt.Sprintf("📦 Всего пакетов: %d\n", stats.TotalPackages))
	output.WriteString(fmt.Sprintf("⬇️ Всего загрузок: %d\n", stats.TotalDownloads))
	output.WriteString(fmt.Sprintf("🕒 Последнее обновление: %s\n\n", stats.LastUpdated.Format("2006-01-02 15:04:05")))

	if len(stats.PopularPackages) > 0 {
		output.WriteString("🔥 Популярные пакеты:\n")
		for i, pkg := range stats.PopularPackages {
			if i >= 10 { // Показываем только топ-10
				break
			}
			output.WriteString(fmt.Sprintf("   %d. %s\n", i+1, pkg))
		}
		output.WriteString("\n")
	}

	if len(stats.PackagesByLicense) > 0 {
		output.WriteString("📜 Распределение по лицензиям:\n")
		for license, count := range stats.PackagesByLicense {
			output.WriteString(fmt.Sprintf("   • %s: %d пакетов\n", license, count))
		}
		output.WriteString("\n")
	}

	if len(stats.PackagesByAuthor) > 0 {
		output.WriteString("👥 Топ авторы:\n")
		// Преобразуем в слайс для сортировки
		type authorStat struct {
			name  string
			count int
		}
		var authors []authorStat
		for author, count := range stats.PackagesByAuthor {
			authors = append(authors, authorStat{author, count})
		}
		// Сортируем по количеству пакетов
		for i := 0; i < len(authors)-1; i++ {
			for j := i + 1; j < len(authors); j++ {
				if authors[i].count < authors[j].count {
					authors[i], authors[j] = authors[j], authors[i]
				}
			}
		}
		// Показываем топ-5 авторов
		for i, author := range authors {
			if i >= 5 {
				break
			}
			output.WriteString(fmt.Sprintf("   %d. %s: %d пакетов\n", i+1, author.name, author.count))
		}
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: output.String(),
		}},
	}, nil
}
