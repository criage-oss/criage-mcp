# Makefile для Criage MCP Server

# Переменные
BINARY_NAME=criage-mcp-server
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
VERSION=1.0.0

# Основная цель
.DEFAULT_GOAL := build

# Сборка для текущей платформы
build:
	@echo "Сборка $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Сборка для Windows
build-windows:
	@echo "Сборка для Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS) .

# Сборка для Linux
build-linux:
	@echo "Сборка для Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_UNIX) .

# Сборка для всех платформ
build-all: build-windows build-linux
	@echo "Сборка завершена для всех платформ"

# Очистка
clean:
	@echo "Очистка файлов сборки..."
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)

# Тестирование
test:
	@echo "Запуск тестов..."
	go test -v ./...

# Форматирование кода
fmt:
	@echo "Форматирование кода..."
	go fmt ./...

# Проверка зависимостей
deps:
	@echo "Проверка зависимостей..."
	go mod tidy
	go mod verify

# Установка
install: build
	@echo "Установка $(BINARY_NAME)..."
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Запуск
run: build
	@echo "Запуск $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Создание релиза
release: clean deps fmt test build-all
	@echo "Создание релиза v$(VERSION)..."
	mkdir -p release
	cp $(BINARY_WINDOWS) release/
	cp $(BINARY_UNIX) release/
	cp README.md release/
	cp config.example.json release/
	@echo "Релиз готов в папке release/"

# Проверка кода
lint:
	@echo "Проверка кода..."
	go vet ./...
	golint ./...

# Справка
help:
	@echo "Доступные команды:"
	@echo "  build        - Сборка для текущей платформы"
	@echo "  build-windows - Сборка для Windows"
	@echo "  build-linux  - Сборка для Linux"
	@echo "  build-all    - Сборка для всех платформ"
	@echo "  clean        - Очистка файлов сборки"
	@echo "  test         - Запуск тестов"
	@echo "  fmt          - Форматирование кода"
	@echo "  deps         - Проверка зависимостей"
	@echo "  install      - Установка в GOPATH/bin"
	@echo "  run          - Сборка и запуск"
	@echo "  release      - Создание релиза"
	@echo "  lint         - Проверка кода"
	@echo "  help         - Эта справка"

.PHONY: build build-windows build-linux build-all clean test fmt deps install run release lint help 