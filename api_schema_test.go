package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// TestRepositoryPackageStructure проверяет соответствие структуры RepositoryPackage схеме API
func TestRepositoryPackageStructure(t *testing.T) {
	now := time.Now()
	repoPackage := RepositoryPackage{
		Name:          "test-package",
		Description:   "Test package description",
		Author:        "Test Author",
		License:       "MIT",
		Homepage:      "https://example.com",
		Repository:    "https://github.com/example/test",
		Keywords:      []string{"test", "example"},
		Versions:      []RepositoryVersion{},
		LatestVersion: "1.0.0",
		Downloads:     100,
		Updated:       now,
	}

	// Проверяем сериализацию/десериализацию
	data, err := json.Marshal(repoPackage)
	if err != nil {
		t.Fatalf("Failed to marshal RepositoryPackage: %v", err)
	}

	var unmarshaled RepositoryPackage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RepositoryPackage: %v", err)
	}

	if unmarshaled.Name != repoPackage.Name {
		t.Errorf("Name field mismatch: expected %s, got %s", repoPackage.Name, unmarshaled.Name)
	}

	if len(unmarshaled.Keywords) != len(repoPackage.Keywords) {
		t.Errorf("Keywords length mismatch: expected %d, got %d", len(repoPackage.Keywords), len(unmarshaled.Keywords))
	}
}

// TestRepositoryVersionStructure проверяет соответствие структуры RepositoryVersion схеме API
func TestRepositoryVersionStructure(t *testing.T) {
	now := time.Now()
	repoVersion := RepositoryVersion{
		Version:      "1.0.0",
		Description:  "Initial version",
		Dependencies: map[string]string{"dep1": "^1.0.0"},
		DevDeps:      map[string]string{"devdep1": "^2.0.0"},
		Files:        []RepositoryFile{},
		Size:         1024,
		Checksum:     "sha256:abcd1234",
		Uploaded:     now,
		Downloads:    50,
	}

	// Проверяем сериализацию/десериализацию
	data, err := json.Marshal(repoVersion)
	if err != nil {
		t.Fatalf("Failed to marshal RepositoryVersion: %v", err)
	}

	var unmarshaled RepositoryVersion
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RepositoryVersion: %v", err)
	}

	if unmarshaled.Version != repoVersion.Version {
		t.Errorf("Version field mismatch: expected %s, got %s", repoVersion.Version, unmarshaled.Version)
	}

	if unmarshaled.Size != repoVersion.Size {
		t.Errorf("Size field mismatch: expected %d, got %d", repoVersion.Size, unmarshaled.Size)
	}
}

// TestRepositoryFileStructure проверяет соответствие структуры RepositoryFile схеме API
func TestRepositoryFileStructure(t *testing.T) {
	repoFile := RepositoryFile{
		OS:       "linux",
		Arch:     "amd64",
		Format:   "tar.zst",
		Filename: "test-package-1.0.0-linux-amd64.tar.zst",
		Size:     2048,
		Checksum: "sha256:efgh5678",
	}

	// Проверяем сериализацию/десериализацию
	data, err := json.Marshal(repoFile)
	if err != nil {
		t.Fatalf("Failed to marshal RepositoryFile: %v", err)
	}

	var unmarshaled RepositoryFile
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RepositoryFile: %v", err)
	}

	if unmarshaled.OS != repoFile.OS {
		t.Errorf("OS field mismatch: expected %s, got %s", repoFile.OS, unmarshaled.OS)
	}

	if unmarshaled.Arch != repoFile.Arch {
		t.Errorf("Arch field mismatch: expected %s, got %s", repoFile.Arch, unmarshaled.Arch)
	}
}

// TestRepositoryStructure проверяет унифицированную структуру Repository
func TestRepositoryStructure(t *testing.T) {
	repo := Repository{
		Name:      "test-repo",
		URL:       "https://packages.example.com",
		Priority:  100,
		Enabled:   true,
		AuthToken: "secret-token",
	}

	// Проверяем сериализацию/десериализацию
	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Failed to marshal Repository: %v", err)
	}

	var unmarshaled Repository
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Repository: %v", err)
	}

	if unmarshaled.Name != repo.Name {
		t.Errorf("Name field mismatch: expected %s, got %s", repo.Name, unmarshaled.Name)
	}

	if unmarshaled.AuthToken != repo.AuthToken {
		t.Errorf("AuthToken field mismatch: expected %s, got %s", repo.AuthToken, unmarshaled.AuthToken)
	}
}

// TestPackageListResponseStructure проверяет новую структуру для списка пакетов
func TestPackageListResponseStructure(t *testing.T) {
	packageList := PackageListResponse{
		Packages: []*RepositoryPackage{
			{
				Name:          "test1",
				LatestVersion: "1.0.0",
			},
			{
				Name:          "test2",
				LatestVersion: "2.0.0",
			},
		},
		Total:      100,
		Page:       1,
		Limit:      20,
		TotalPages: 5,
	}

	// Проверяем сериализацию/десериализацию
	data, err := json.Marshal(packageList)
	if err != nil {
		t.Fatalf("Failed to marshal PackageListResponse: %v", err)
	}

	var unmarshaled PackageListResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal PackageListResponse: %v", err)
	}

	if len(unmarshaled.Packages) != len(packageList.Packages) {
		t.Errorf("Packages length mismatch: expected %d, got %d", len(packageList.Packages), len(unmarshaled.Packages))
	}

	if unmarshaled.Total != packageList.Total {
		t.Errorf("Total field mismatch: expected %d, got %d", packageList.Total, unmarshaled.Total)
	}
}

// TestApiSchemaCompatibility проверяет совместимость с API схемой из документации
func TestApiSchemaCompatibility(t *testing.T) {
	// Проверяем, что все основные типы данных имеют правильные JSON теги
	testCases := []struct {
		name       string
		structType reflect.Type
		fieldTests map[string]string // поле -> ожидаемый JSON тег
	}{
		{
			name:       "RepositoryPackage",
			structType: reflect.TypeOf(RepositoryPackage{}),
			fieldTests: map[string]string{
				"Name":          "name",
				"Description":   "description",
				"Author":        "author",
				"License":       "license",
				"LatestVersion": "latest_version",
				"Downloads":     "downloads",
			},
		},
		{
			name:       "RepositoryVersion",
			structType: reflect.TypeOf(RepositoryVersion{}),
			fieldTests: map[string]string{
				"Version":      "version",
				"Dependencies": "dependencies",
				"DevDeps":      "dev_dependencies",
				"Size":         "size",
				"Checksum":     "checksum",
				"Downloads":    "downloads",
			},
		},
		{
			name:       "RepositoryFile",
			structType: reflect.TypeOf(RepositoryFile{}),
			fieldTests: map[string]string{
				"OS":       "os",
				"Arch":     "arch",
				"Format":   "format",
				"Filename": "filename",
				"Size":     "size",
				"Checksum": "checksum",
			},
		},
		{
			name:       "Repository",
			structType: reflect.TypeOf(Repository{}),
			fieldTests: map[string]string{
				"Name":      "name",
				"URL":       "url",
				"Priority":  "priority",
				"Enabled":   "enabled",
				"AuthToken": "auth_token,omitempty",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for fieldName, expectedTag := range tc.fieldTests {
				field, found := tc.structType.FieldByName(fieldName)
				if !found {
					t.Errorf("Field %s not found in %s", fieldName, tc.name)
					continue
				}

				jsonTag := field.Tag.Get("json")
				if jsonTag != expectedTag {
					t.Errorf("Field %s in %s has wrong JSON tag: expected %s, got %s", fieldName, tc.name, expectedTag, jsonTag)
				}
			}
		})
	}
}

// TestRateLimiterFunctionality проверяет работу rate limiter
func TestRateLimiterFunctionality(t *testing.T) {
	// Создаем rate limiter с высокой частотой для быстрого тестирования
	rl := NewRateLimiter(100) // 100 запросов в секунду
	defer rl.Close()

	// Проверяем, что rate limiter не блокирует нормальные запросы
	start := time.Now()
	for i := 0; i < 5; i++ {
		rl.Wait()
	}
	elapsed := time.Since(start)

	// Должно занимать меньше секунды для 5 запросов при лимите 100/сек
	if elapsed > time.Second {
		t.Errorf("Rate limiter is too slow: took %v for 5 requests", elapsed)
	}

	// Проверяем, что rate limiter действительно ограничивает частоту
	rl2 := NewRateLimiter(2) // 2 запроса в секунду
	defer rl2.Close()

	start = time.Now()
	for i := 0; i < 3; i++ {
		rl2.Wait()
	}
	elapsed = time.Since(start)

	// Должно занимать как минимум 1 секунду для 3 запросов при лимите 2/сек
	if elapsed < time.Second {
		t.Errorf("Rate limiter is not working: took only %v for 3 requests with 2/sec limit", elapsed)
	}
}

// BenchmarkRateLimiter бенчмарк для rate limiter
func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(1000) // 1000 запросов в секунду
	defer rl.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Wait()
	}
}

// TestNewApiEndpoints проверяет новые эндпоинты API
func TestNewApiEndpoints(t *testing.T) {
	pm, err := NewPackageManager()
	if err != nil {
		t.Skipf("Failed to create PackageManager: %v", err)
	}

	// Проверяем, что методы существуют (компиляция пройдет только если методы определены)
	// Вызываем методы с пустыми параметрами для проверки их наличия
	_, err = pm.ListRepositoryPackages("", 1, 10)
	if err == nil {
		t.Log("ListRepositoryPackages method is available")
	}

	_, err = pm.GetPackageVersionInfo("", "", "")
	if err == nil {
		t.Log("GetPackageVersionInfo method is available")
	}

	// Проверяем, что rate limiter инициализирован
	if pm.rateLimiter == nil {
		t.Error("Rate limiter is not initialized")
	}
}

// TestUnifiedTokenField проверяет унификацию поля токена
func TestUnifiedTokenField(t *testing.T) {
	repo := Repository{
		Name:      "test",
		URL:       "https://example.com",
		AuthToken: "test-token",
	}

	// Проверяем, что поле называется AuthToken, а не Token
	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Failed to marshal Repository: %v", err)
	}

	// Проверяем, что в JSON используется auth_token
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := jsonMap["auth_token"]; !exists {
		t.Error("JSON should contain 'auth_token' field")
	}

	if _, exists := jsonMap["token"]; exists {
		t.Error("JSON should not contain old 'token' field")
	}
}
