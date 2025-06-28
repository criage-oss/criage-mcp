package main

import (
	"time"
)

// PackageInfo информация об установленном пакете
type PackageInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	InstallDate  time.Time         `json:"install_date"`
	InstallPath  string            `json:"install_path"`
	Global       bool              `json:"global"`
	Dependencies map[string]string `json:"dependencies"`
	Size         int64             `json:"size"`
	Files        []string          `json:"files"`
	Scripts      map[string]string `json:"scripts"`
}

// SearchResult результат поиска пакетов
type SearchResult struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Downloads   int64     `json:"downloads"`
	Updated     time.Time `json:"updated"`
	Score       float64   `json:"score"`
}

// PackageManifest манифест пакета
type PackageManifest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	License      string                 `json:"license"`
	Homepage     string                 `json:"homepage"`
	Repository   string                 `json:"repository"`
	Keywords     []string               `json:"keywords"`
	Dependencies map[string]string      `json:"dependencies"`
	DevDeps      map[string]string      `json:"dev_dependencies"`
	Files        []string               `json:"files"`
	Scripts      map[string]string      `json:"scripts"`
	Hooks        *PackageHooks          `json:"hooks"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// PackageHooks хуки пакета
type PackageHooks struct {
	PreInstall  []string `json:"pre_install"`
	PostInstall []string `json:"post_install"`
	PreRemove   []string `json:"pre_remove"`
	PostRemove  []string `json:"post_remove"`
}

// Config конфигурация пакетного менеджера
type Config struct {
	Repositories     []Repository `json:"repositories"`
	GlobalPath       string       `json:"global_path"`
	LocalPath        string       `json:"local_path"`
	CachePath        string       `json:"cache_path"`
	TempPath         string       `json:"temp_path"`
	Timeout          int          `json:"timeout"`
	MaxConcurrency   int          `json:"max_concurrency"`
	CompressionLevel int          `json:"compression_level"`
	ForceHTTPS       bool         `json:"force_https"`
}

// Repository репозиторий пакетов
type Repository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
	Token    string `json:"token,omitempty"`
}

// RepositoryPackage информация о пакете в репозитории
type RepositoryPackage struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Author      string              `json:"author"`
	License     string              `json:"license"`
	Homepage    string              `json:"homepage"`
	Repository  string              `json:"repository"`
	Keywords    []string            `json:"keywords"`
	Versions    []RepositoryVersion `json:"versions"`
	Downloads   int64               `json:"downloads"`
	Updated     time.Time           `json:"updated"`
}

// RepositoryVersion версия пакета в репозитории
type RepositoryVersion struct {
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Dependencies map[string]string `json:"dependencies"`
	DevDeps      map[string]string `json:"dev_dependencies"`
	Files        []RepositoryFile  `json:"files"`
	Size         int64             `json:"size"`
	Checksum     string            `json:"checksum"`
	Uploaded     time.Time         `json:"uploaded"`
	Downloads    int64             `json:"downloads"`
}

// RepositoryFile файл пакета для разных платформ
type RepositoryFile struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Format   string `json:"format"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	URL      string `json:"url"`
}

// BuildManifest манифест сборки
type BuildManifest struct {
	BuildScript string              `json:"build_script"`
	OutputDir   string              `json:"output_dir"`
	Targets     []BuildTarget       `json:"targets"`
	Compression CompressionSettings `json:"compression"`
}

// BuildTarget целевая платформа
type BuildTarget struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// CompressionSettings настройки сжатия
type CompressionSettings struct {
	Format string `json:"format"`
	Level  int    `json:"level"`
}

// ArchiveMetadata метаданные архива
type ArchiveMetadata struct {
	CompressionType string           `json:"compression_type"`
	CreatedAt       string           `json:"created_at"`
	CreatedBy       string           `json:"created_by"`
	PackageManifest *PackageManifest `json:"package_manifest,omitempty"`
	BuildManifest   *BuildManifest   `json:"build_manifest,omitempty"`
}

// Statistics статистика репозитория
type Statistics struct {
	TotalDownloads    int64          `json:"total_downloads"`
	PackagesByLicense map[string]int `json:"packages_by_license"`
	PackagesByAuthor  map[string]int `json:"packages_by_author"`
	PopularPackages   []string       `json:"popular_packages"`
	LastUpdated       time.Time      `json:"last_updated"`
	TotalPackages     int            `json:"total_packages"`
}
