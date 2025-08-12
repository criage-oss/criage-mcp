package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

// RateLimiter простой rate limiter для HTTP запросов
type RateLimiter struct {
	ticker   *time.Ticker
	requests chan struct{}
}

// NewRateLimiter создает новый rate limiter с заданной частотой запросов в секунду
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 10 // по умолчанию 10 запросов в секунду
	}

	interval := time.Second / time.Duration(requestsPerSecond)
	ticker := time.NewTicker(interval)
	requests := make(chan struct{}, 1)
	// Стартовое «разрешение»
	requests <- struct{}{}

	rl := &RateLimiter{
		ticker:   ticker,
		requests: requests,
	}

	// Запускаем горутину для пополнения буфера
	go func() {
		for range ticker.C {
			// Тик добавляет одно «разрешение», не накапливая больше одного
			select {
			case requests <- struct{}{}:
			default:
			}
		}
	}()

	return rl
}

// Wait ждет разрешения на выполнение запроса
func (rl *RateLimiter) Wait() {
	<-rl.requests
}

// Close останавливает rate limiter
func (rl *RateLimiter) Close() {
	rl.ticker.Stop()
	close(rl.requests)
}

// PackageManager основной менеджер пакетов
type PackageManager struct {
	config            *Config
	installedPackages map[string]*PackageInfo
	packagesMutex     sync.RWMutex
	httpClient        *http.Client
	rateLimiter       *RateLimiter
}

// NewPackageManager создает новый пакетный менеджер
func NewPackageManager() (*PackageManager, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	pm := &PackageManager{
		config:            config,
		installedPackages: make(map[string]*PackageInfo),
		httpClient:        httpClient,
		rateLimiter:       NewRateLimiter(5), // 5 запросов в секунду
	}

	// Создаем необходимые директории
	if err := pm.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("ошибка создания директорий: %w", err)
	}

	// Загружаем информацию об установленных пакетах
	if err := pm.loadInstalledPackages(); err != nil {
		return nil, fmt.Errorf("ошибка загрузки установленных пакетов: %w", err)
	}

	return pm, nil
}

// loadConfig загружает конфигурацию
func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".criage", "config.json")

	// Создаем конфигурацию по умолчанию
	config := &Config{
		Repositories: []Repository{
			{
				Name:     "criage-main",
				URL:      "https://packages.criage.ru",
				Priority: 1,
				Enabled:  true,
			},
		},
		GlobalPath:       filepath.Join(homeDir, ".criage", "packages"),
		LocalPath:        "./criage_modules",
		CachePath:        filepath.Join(homeDir, ".criage", "cache"),
		TempPath:         filepath.Join(homeDir, ".criage", "temp"),
		Timeout:          30,
		MaxConcurrency:   4,
		CompressionLevel: 3,
		ForceHTTPS:       false,
	}

	// Если файл конфигурации существует, загружаем его
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	} else {
		// Создаем файл конфигурации по умолчанию
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, err
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// ensureDirectories создает необходимые директории
func (pm *PackageManager) ensureDirectories() error {
	dirs := []string{
		pm.config.GlobalPath,
		pm.config.LocalPath,
		pm.config.CachePath,
		pm.config.TempPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// InstallPackage устанавливает пакет
func (pm *PackageManager) InstallPackage(packageName, version string, global, force, dev bool, arch, osName string) error {
	// Проверяем, не установлен ли уже пакет
	if !force {
		if info, exists := pm.getInstalledPackage(packageName); exists {
			if version == "" || info.Version == version {
				return fmt.Errorf("пакет %s (%s) уже установлен", packageName, info.Version)
			}
		}
	}

	// Определяем архитектуру и ОС
	if arch == "" {
		arch = runtime.GOARCH
	}
	if osName == "" {
		osName = runtime.GOOS
	}

	// Поиск пакета в репозиториях
	packageInfo, downloadURL, err := pm.findPackage(packageName, version, arch, osName)
	if err != nil {
		return fmt.Errorf("пакет не найден: %w", err)
	}

	// Скачиваем пакет
	archivePath, err := pm.downloadPackage(downloadURL, packageName, packageInfo.Version)
	if err != nil {
		return fmt.Errorf("ошибка скачивания: %w", err)
	}
	defer os.Remove(archivePath)

	// Извлекаем архив
	tempDir := filepath.Join(pm.config.TempPath, fmt.Sprintf("install_%s_%d", packageName, time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	if err := pm.extractArchive(archivePath, tempDir); err != nil {
		return fmt.Errorf("ошибка извлечения: %w", err)
	}

	// Загружаем манифест пакета
	manifest, err := pm.loadManifestFromDir(tempDir)
	if err != nil {
		return fmt.Errorf("ошибка загрузки манифеста: %w", err)
	}

	// Определяем путь установки
	installPath := pm.getInstallPath(packageName, global)

	// Удаляем старую версию, если она есть
	if force {
		if err := os.RemoveAll(installPath); err != nil {
			return fmt.Errorf("ошибка удаления старой версии: %w", err)
		}
	}

	// Создаем директорию установки
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Копируем файлы
	if err := pm.copyFiles(tempDir, installPath); err != nil {
		return fmt.Errorf("ошибка копирования файлов: %w", err)
	}

	// Создаем информацию о пакете
	packageInfo = &PackageInfo{
		Name:         manifest.Name,
		Version:      manifest.Version,
		Description:  manifest.Description,
		Author:       manifest.Author,
		License:      manifest.License,
		InstallDate:  time.Now(),
		InstallPath:  installPath,
		Global:       global,
		Dependencies: manifest.Dependencies,
		Size:         pm.calculateDirSize(installPath),
		Files:        manifest.Files,
		Scripts:      manifest.Scripts,
	}

	// Сохраняем информацию о пакете
	if err := pm.savePackageInfo(packageInfo); err != nil {
		return fmt.Errorf("ошибка сохранения информации о пакете: %w", err)
	}

	// Обновляем кеш установленных пакетов
	pm.packagesMutex.Lock()
	pm.installedPackages[packageName] = packageInfo
	pm.packagesMutex.Unlock()

	return nil
}

// UninstallPackage удаляет пакет
func (pm *PackageManager) UninstallPackage(packageName string, global, purge bool) error {
	// Проверяем, установлен ли пакет
	packageInfo, exists := pm.getInstalledPackage(packageName)
	if !exists {
		return fmt.Errorf("пакет %s не установлен", packageName)
	}

	// Удаляем файлы пакета
	if err := os.RemoveAll(packageInfo.InstallPath); err != nil {
		return fmt.Errorf("ошибка удаления файлов: %w", err)
	}

	// Удаляем информацию о пакете
	if err := pm.removePackageInfo(packageName, global); err != nil {
		return fmt.Errorf("ошибка удаления информации о пакете: %w", err)
	}

	// Обновляем кеш
	pm.packagesMutex.Lock()
	delete(pm.installedPackages, packageName)
	pm.packagesMutex.Unlock()

	return nil
}

// UpdatePackage обновляет пакет
func (pm *PackageManager) UpdatePackage(packageName string) error {
	// Проверяем, установлен ли пакет
	currentInfo, exists := pm.getInstalledPackage(packageName)
	if !exists {
		return fmt.Errorf("пакет %s не установлен", packageName)
	}

	// Ищем последнюю версию
	latestInfo, _, err := pm.findPackage(packageName, "", runtime.GOARCH, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("не удалось найти обновления: %w", err)
	}

	// Проверяем, нужно ли обновление
	if currentInfo.Version == latestInfo.Version {
		return fmt.Errorf("пакет %s уже имеет последнюю версию (%s)", packageName, currentInfo.Version)
	}

	// Устанавливаем новую версию
	return pm.InstallPackage(packageName, latestInfo.Version, currentInfo.Global, true, false, "", "")
}

// SearchPackages выполняет поиск пакетов
func (pm *PackageManager) SearchPackages(query string) ([]SearchResult, error) {
	var allResults []SearchResult

	for _, repo := range pm.config.Repositories {
		if !repo.Enabled {
			continue
		}

		results, err := pm.searchInRepository(repo, query)
		if err != nil {
			continue // Игнорируем ошибки отдельных репозиториев
		}

		allResults = append(allResults, results...)
	}

	// Сортируем по релевантности
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	return allResults, nil
}

// ListPackages возвращает список установленных пакетов
func (pm *PackageManager) ListPackages(global, outdated bool) ([]*PackageInfo, error) {
	pm.packagesMutex.RLock()
	defer pm.packagesMutex.RUnlock()

	var packages []*PackageInfo
	for _, pkg := range pm.installedPackages {
		if global && !pkg.Global {
			continue
		}
		if !global && pkg.Global {
			continue
		}

		packages = append(packages, pkg)
	}

	// Сортируем по имени
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return packages, nil
}

// GetPackageInfo возвращает информацию о пакете
func (pm *PackageManager) GetPackageInfo(packageName string) (*PackageInfo, error) {
	info, exists := pm.getInstalledPackage(packageName)
	if !exists {
		return nil, fmt.Errorf("пакет %s не установлен", packageName)
	}
	return info, nil
}

// CreatePackage создает новый пакет
func (pm *PackageManager) CreatePackage(name, template, author, description string) error {
	// Создаем директорию для нового пакета
	packageDir := filepath.Join(".", name)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Создаем манифест пакета
	manifest := &PackageManifest{
		Name:         name,
		Version:      "0.1.0",
		Description:  description,
		Author:       author,
		License:      "MIT",
		Keywords:     []string{},
		Dependencies: make(map[string]string),
		DevDeps:      make(map[string]string),
		Files:        []string{"src/"},
		Scripts:      make(map[string]string),
	}

	// Сохраняем манифест
	manifestPath := filepath.Join(packageDir, "criage.yaml")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка кодирования манифеста: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения манифеста: %w", err)
	}

	// Создаем базовую структуру
	srcDir := filepath.Join(packageDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания src директории: %w", err)
	}

	// Создаем README
	readmePath := filepath.Join(packageDir, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\n%s\n\n## Установка\n\n```bash\ncriage install %s\n```\n", name, description, name)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("ошибка создания README: %w", err)
	}

	return nil
}

// BuildPackage собирает пакет
func (pm *PackageManager) BuildPackage(outputPath, format string, compressionLevel int) error {
	// Загружаем манифест
	manifest, err := pm.loadManifestFromDir(".")
	if err != nil {
		return fmt.Errorf("ошибка загрузки манифеста: %w", err)
	}

	// Определяем выходной файл
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s-%s.%s", manifest.Name, manifest.Version, format)
	}

	// Создаем архив
	if err := pm.createArchive(".", outputPath, format, compressionLevel); err != nil {
		return fmt.Errorf("ошибка создания архива: %w", err)
	}

	return nil
}

// PublishPackage публикует пакет в репозиторий
func (pm *PackageManager) PublishPackage(registryURL, token string) error {
	// Загружаем манифест
	manifest, err := pm.loadManifestFromDir(".")
	if err != nil {
		return fmt.Errorf("ошибка загрузки манифеста: %w", err)
	}

	// Строим пакет
	archivePath := fmt.Sprintf("%s-%s.criage", manifest.Name, manifest.Version)
	if err := pm.BuildPackage(archivePath, "criage", pm.config.CompressionLevel); err != nil {
		return fmt.Errorf("ошибка сборки пакета: %w", err)
	}
	defer os.Remove(archivePath)

	// Загружаем в репозиторий
	if registryURL == "" {
		registryURL = pm.config.Repositories[0].URL
	}

	return pm.uploadPackage(registryURL, archivePath, token)
}

// Вспомогательные методы

func (pm *PackageManager) getInstalledPackage(packageName string) (*PackageInfo, bool) {
	pm.packagesMutex.RLock()
	defer pm.packagesMutex.RUnlock()
	info, exists := pm.installedPackages[packageName]
	return info, exists
}

func (pm *PackageManager) findPackage(packageName, version, arch, osName string) (*PackageInfo, string, error) {
	for _, repo := range pm.config.Repositories {
		if !repo.Enabled {
			continue
		}

		info, url, err := pm.findInRepository(repo, packageName, version, arch, osName)
		if err == nil {
			return info, url, nil
		}
	}

	return nil, "", fmt.Errorf("пакет %s не найден", packageName)
}

func (pm *PackageManager) findInRepository(repo Repository, packageName, version, arch, osName string) (*PackageInfo, string, error) {
	// Получаем информацию о пакете из репозитория
	url := fmt.Sprintf("%s/api/v1/packages/%s", repo.URL, packageName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	if repo.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+repo.AuthToken)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("ошибка получения информации о пакете: %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool               `json:"success"`
		Data    *RepositoryPackage `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, "", err
	}

	if !apiResp.Success || apiResp.Data == nil {
		return nil, "", fmt.Errorf("пакет не найден в репозитории")
	}

	pkg := apiResp.Data

	// Выбираем версию
	var selectedVersion *RepositoryVersion
	if version == "" {
		// Берем последнюю версию
		if len(pkg.Versions) > 0 {
			selectedVersion = &pkg.Versions[len(pkg.Versions)-1]
		}
	} else {
		// Ищем указанную версию
		for _, v := range pkg.Versions {
			if v.Version == version {
				selectedVersion = &v
				break
			}
		}
	}

	if selectedVersion == nil {
		return nil, "", fmt.Errorf("версия %s не найдена", version)
	}

	// Ищем подходящий файл
	var selectedFile *RepositoryFile
	for _, file := range selectedVersion.Files {
		if file.OS == osName && file.Arch == arch {
			selectedFile = &file
			break
		}
	}

	if selectedFile == nil {
		return nil, "", fmt.Errorf("файл для %s/%s не найден", osName, arch)
	}

	info := &PackageInfo{
		Name:        pkg.Name,
		Version:     selectedVersion.Version,
		Description: pkg.Description,
		Author:      pkg.Author,
		License:     pkg.License,
		Size:        selectedFile.Size,
	}

	// Строим URL для скачивания на основе информации о файле
	downloadURL := fmt.Sprintf("%s/api/v1/download/%s/%s/%s",
		repo.URL, pkg.Name, selectedVersion.Version, selectedFile.Filename)

	return info, downloadURL, nil
}

func (pm *PackageManager) downloadPackage(url, packageName, version string) (string, error) {
	resp, err := pm.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка скачивания: %d", resp.StatusCode)
	}

	// Создаем временный файл
	tempFile := filepath.Join(pm.config.TempPath, fmt.Sprintf("%s-%s.tmp", packageName, version))

	file, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Копируем данные
	if _, err := io.Copy(file, resp.Body); err != nil {
		os.Remove(tempFile)
		return "", err
	}

	return tempFile, nil
}

func (pm *PackageManager) loadInstalledPackages() error {
	// Загружаем глобальные пакеты
	globalInfoPath := filepath.Join(pm.config.GlobalPath, "packages.json")
	if err := pm.loadPackagesFromFile(globalInfoPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Загружаем локальные пакеты
	localInfoPath := filepath.Join(pm.config.LocalPath, "packages.json")
	if err := pm.loadPackagesFromFile(localInfoPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (pm *PackageManager) loadPackagesFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var packages map[string]*PackageInfo
	if err := json.Unmarshal(data, &packages); err != nil {
		return err
	}

	pm.packagesMutex.Lock()
	defer pm.packagesMutex.Unlock()

	for name, info := range packages {
		pm.installedPackages[name] = info
	}

	return nil
}

func (pm *PackageManager) savePackageInfo(info *PackageInfo) error {
	var packagesPath string
	if info.Global {
		packagesPath = filepath.Join(pm.config.GlobalPath, "packages.json")
	} else {
		packagesPath = filepath.Join(pm.config.LocalPath, "packages.json")
	}

	// Загружаем существующие пакеты
	var packages map[string]*PackageInfo
	if data, err := os.ReadFile(packagesPath); err == nil {
		json.Unmarshal(data, &packages)
	}
	if packages == nil {
		packages = make(map[string]*PackageInfo)
	}

	// Добавляем новый пакет
	packages[info.Name] = info

	// Сохраняем
	data, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(packagesPath, data, 0644)
}

func (pm *PackageManager) removePackageInfo(packageName string, global bool) error {
	var packagesPath string
	if global {
		packagesPath = filepath.Join(pm.config.GlobalPath, "packages.json")
	} else {
		packagesPath = filepath.Join(pm.config.LocalPath, "packages.json")
	}

	// Загружаем существующие пакеты
	var packages map[string]*PackageInfo
	if data, err := os.ReadFile(packagesPath); err == nil {
		json.Unmarshal(data, &packages)
	}
	if packages == nil {
		return nil
	}

	// Удаляем пакет
	delete(packages, packageName)

	// Сохраняем
	data, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(packagesPath, data, 0644)
}

func (pm *PackageManager) getInstallPath(packageName string, global bool) string {
	if global {
		return filepath.Join(pm.config.GlobalPath, packageName)
	}
	return filepath.Join(pm.config.LocalPath, packageName)
}

func (pm *PackageManager) extractArchive(archivePath, destPath string) error {
	// Простая заглушка для извлечения архивов
	// В реальной реализации здесь должна быть логика для разных форматов
	return fmt.Errorf("извлечение архивов пока не реализовано")
}

func (pm *PackageManager) copyFiles(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

func (pm *PackageManager) loadManifestFromDir(dir string) (*PackageManifest, error) {
	manifestPath := filepath.Join(dir, "criage.yaml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (pm *PackageManager) calculateDirSize(dir string) int64 {
	var size int64

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size
}

func (pm *PackageManager) searchInRepository(repo Repository, query string) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/api/v1/search?q=%s", repo.URL, query)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if repo.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+repo.AuthToken)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка поиска: %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			Query   string         `json:"query"`
			Results []SearchResult `json:"results"`
			Total   int            `json:"total"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("ошибка поиска в репозитории")
	}

	return apiResp.Data.Results, nil
}

func (pm *PackageManager) createArchive(srcDir, outputPath, format string, compressionLevel int) error {
	// Заглушка для создания архивов
	return fmt.Errorf("создание архивов пока не реализовано")
}

func (pm *PackageManager) uploadPackage(registryURL, archivePath, token string) error {
	// Открываем файл для загрузки
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	// Создаем multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Добавляем файл в form
	part, err := writer.CreateFormFile("package", filepath.Base(archivePath))
	if err != nil {
		return fmt.Errorf("ошибка создания form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("ошибка копирования файла: %w", err)
	}

	writer.Close()

	// Создаем POST запрос
	uploadURL := fmt.Sprintf("%s/api/v1/upload", registryURL)
	req, err := http.NewRequest("POST", uploadURL, &body)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	// Выполняем запрос
	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("неверный токен авторизации")
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("ошибка сервера: %d", resp.StatusCode)
	}

	// Читаем ответ
	var result struct {
		Success  bool   `json:"success"`
		Message  string `json:"message"`
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("операция не удалась: %s", result.Message)
	}

	return nil
}

func (pm *PackageManager) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// RefreshRepositoryIndex принудительно обновляет индекс пакетов в репозитории
func (pm *PackageManager) RefreshRepositoryIndex(repositoryURL, authToken string) error {
	// Создаем URL для эндпоинта обновления индекса
	refreshURL := fmt.Sprintf("%s/api/v1/refresh", repositoryURL)

	// Создаем POST запрос
	req, err := http.NewRequest("POST", refreshURL, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем токен авторизации
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	// Выполняем запрос
	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("неверный токен авторизации")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка сервера: %d", resp.StatusCode)
	}

	// Читаем ответ
	var result struct {
		Success       bool   `json:"success"`
		Message       string `json:"message"`
		TotalPackages int    `json:"total_packages"`
		LastUpdated   string `json:"last_updated"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("операция не удалась: %s", result.Message)
	}

	return nil
}

// GetRepositoryStats получает детальную статистику репозитория
func (pm *PackageManager) GetRepositoryStats(repositoryURL string) (*Statistics, error) {
	// Создаем URL для эндпоинта статистики
	statsURL := fmt.Sprintf("%s/api/v1/stats", repositoryURL)

	// Создаем GET запрос
	req, err := http.NewRequest("GET", statsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	// Выполняем запрос
	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка сервера: %d", resp.StatusCode)
	}

	// Читаем ответ
	var apiResp struct {
		Success bool        `json:"success"`
		Data    *Statistics `json:"data"`
		Message string      `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("операция не удалась: %s", apiResp.Message)
	}

	if apiResp.Data == nil {
		return nil, fmt.Errorf("пустые данные статистики")
	}

	return apiResp.Data, nil
}

// GetRepositoryInfo получает информацию о репозитории
func (pm *PackageManager) GetRepositoryInfo(repositoryURL string) (map[string]interface{}, error) {
	// Создаем URL для эндпоинта информации о репозитории
	infoURL := fmt.Sprintf("%s/api/v1/", repositoryURL)

	// Создаем GET запрос
	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	// Выполняем запрос
	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка сервера: %d", resp.StatusCode)
	}

	// Читаем ответ
	var apiResp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
		Message string                 `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("операция не удалась: %s", apiResp.Message)
	}

	if apiResp.Data == nil {
		return nil, fmt.Errorf("пустые данные репозитория")
	}

	return apiResp.Data, nil
}

// PackageListResponse структура ответа для списка пакетов с пагинацией
type PackageListResponse struct {
	Packages   []*RepositoryPackage `json:"packages"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

// ListRepositoryPackages получает список всех пакетов из репозитория с пагинацией
func (pm *PackageManager) ListRepositoryPackages(repositoryURL string, page, limit int) (*PackageListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Создаем URL для эндпоинта списка пакетов
	listURL := fmt.Sprintf("%s/api/v1/packages?page=%d&limit=%d", repositoryURL, page, limit)

	// Создаем GET запрос
	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	// Выполняем запрос
	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка сервера: %d", resp.StatusCode)
	}

	// Читаем ответ
	var apiResp struct {
		Success bool                 `json:"success"`
		Data    *PackageListResponse `json:"data"`
		Error   string               `json:"error"`
		Message string               `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != "" {
			return nil, fmt.Errorf("операция не удалась: %s", apiResp.Error)
		}
		return nil, fmt.Errorf("операция не удалась: %s", apiResp.Message)
	}

	if apiResp.Data == nil {
		return nil, fmt.Errorf("пустые данные списка пакетов")
	}

	return apiResp.Data, nil
}

// GetPackageVersionInfo получает информацию о конкретной версии пакета
func (pm *PackageManager) GetPackageVersionInfo(repositoryURL, packageName, version string) (*RepositoryVersion, error) {
	// Создаем URL для эндпоинта конкретной версии пакета
	versionURL := fmt.Sprintf("%s/api/v1/packages/%s/%s", repositoryURL, packageName, version)

	// Создаем GET запрос
	req, err := http.NewRequest("GET", versionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Применяем rate limiting
	pm.rateLimiter.Wait()

	// Выполняем запрос
	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("версия пакета не найдена: %s@%s", packageName, version)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка сервера: %d", resp.StatusCode)
	}

	// Читаем ответ
	var apiResp struct {
		Success bool               `json:"success"`
		Data    *RepositoryVersion `json:"data"`
		Error   string             `json:"error"`
		Message string             `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != "" {
			return nil, fmt.Errorf("операция не удалась: %s", apiResp.Error)
		}
		return nil, fmt.Errorf("операция не удалась: %s", apiResp.Message)
	}

	if apiResp.Data == nil {
		return nil, fmt.Errorf("пустые данные версии пакета")
	}

	return apiResp.Data, nil
}
