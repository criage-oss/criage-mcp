# Criage MCP Server

MCP (Model Context Protocol) сервер для пакетного менеджера Criage. Предоставляет полный функционал клиента Criage через протокол MCP для интеграции с AI инструментами.

## Описание

Этот сервер дублирует весь функционал клиента Criage и предоставляет его через MCP протокол. AI может использовать все возможности пакетного менеджера:

- Установка и удаление пакетов
- Поиск пакетов в репозиториях
- Управление зависимостями
- Создание новых пакетов
- Сборка и публикация пакетов
- Получение информации о пакетах и репозиториях

## Установка

```bash
cd mcp-server
go mod tidy
go build -o criage-mcp-server .
```

## Использование

MCP сервер работает через стандартные потоки ввода/вывода:

```bash
./criage-mcp-server
```

## Доступные инструменты

### Управление пакетами

- `install_package` - Установка пакета из репозитория
- `uninstall_package` - Удаление установленного пакета  
- `update_package` - Обновление пакета до последней версии
- `list_packages` - Список установленных пакетов
- `package_info` - Подробная информация о пакете

### Поиск и исследование

- `search_packages` - Поиск пакетов в репозиториях
- `repository_info` - Информация о репозитории

### Разработка

- `create_package` - Создание нового пакета
- `build_package` - Сборка пакета
- `publish_package` - Публикация пакета в репозиторий

## Конфигурация

Сервер использует конфигурацию из `~/.criage/config.json`. Если файл не существует, создается автоматически с настройками по умолчанию:

```json
{
  "repositories": [
    {
      "name": "default",
      "url": "http://localhost:8080", 
      "priority": 1,
      "enabled": true
    }
  ],
  "global_path": "~/.criage/packages",
  "local_path": "./criage_modules",
  "cache_path": "~/.criage/cache",
  "temp_path": "~/.criage/temp",
  "timeout": 30,
  "max_concurrency": 4,
  "compression_level": 3,
  "force_https": false
}
```

## Примеры использования через MCP

### Установка пакета

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "install_package",
    "arguments": {
      "name": "example-package",
      "version": "1.0.0",
      "global": false
    }
  }
}
```

### Поиск пакетов

```json
{
  "jsonrpc": "2.0", 
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search_packages",
    "arguments": {
      "query": "web framework"
    }
  }
}
```

### Создание нового пакета

```json
{
  "jsonrpc": "2.0",
  "id": 3, 
  "method": "tools/call",
  "params": {
    "name": "create_package",
    "arguments": {
      "name": "my-package",
      "template": "basic",
      "author": "Мой автор",
      "description": "Описание моего пакета"
    }
  }
}
```

## Архитектура

```
mcp-server/
├── main.go              # Основной файл MCP сервера
├── types.go             # Структуры данных
├── package_manager.go   # Пакетный менеджер
├── go.mod              # Go модули
└── README.md           # Документация
```

## Интеграция с AI

Сервер полностью совместим с Claude Desktop и другими MCP клиентами. Для добавления в Claude Desktop:

1. Соберите сервер: `go build -o criage-mcp-server .`
2. Добавьте в конфигурацию Claude Desktop (`config.json`):

```json
{
  "mcpServers": {
    "criage": {
      "command": "/path/to/criage-mcp-server"
    }
  }
}
```

## Совместимость

- Полная совместимость с основным клиентом Criage
- Поддержка всех форматов архивов (criage, tar.zst, tar.lz4, tar.xz, tar.gz, zip)
- Работа с репозиториями Criage
- Мультиплатформенность (Windows, Linux, macOS)

## Лицензия

Использует ту же лицензию, что и основной проект Criage. 