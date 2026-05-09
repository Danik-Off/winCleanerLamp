package cleaner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// JunkRecord описывает записанный элемент мусора.
type JunkRecord struct {
	Path        string    `json:"path"`        // полный путь к файлу/папке
	CategoryID  string    `json:"category_id"` // категория откуда найден
	Size        int64     `json:"size"`        // размер в байтах
	DeletedAt   time.Time `json:"deleted_at"`  // когда был удалён (пусто = ещё существует)
	ProgramHint string    `json:"program_hint"`// подсказка от какой программы (если есть)
}

// JunkConfig — структура конфига учёта мусора.
type JunkConfig struct {
	Version   string            `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Records   []JunkRecord      `json:"records"`
	Stats     map[string]int64  `json:"stats,omitempty"` // category_id -> total_bytes
}

// DefaultConfigPath возвращает путь к файлу конфига.
func DefaultConfigPath() string {
	// %LOCALAPPDATA%\winCleanerLamp\junk.json
	home, err := os.UserHomeDir()
	if err != nil {
		return "junk.json" // fallback
	}
	dir := filepath.Join(home, "AppData", "Local", "winCleanerLamp")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "junk.json")
}

// LoadJunkConfig загружает конфиг из файла.
func LoadJunkConfig(path string) (*JunkConfig, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &JunkConfig{
				Version:   "1.0",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Records:   []JunkRecord{},
				Stats:     map[string]int64{},
			}, nil
		}
		return nil, err
	}
	var cfg JunkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Stats == nil {
		cfg.Stats = map[string]int64{}
	}
	return &cfg, nil
}

// SaveJunkConfig сохраняет конфиг в файл.
func SaveJunkConfig(cfg *JunkConfig, path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}
	cfg.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// Запись в temp файл и переименование для атомарности
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AddRecord добавляет запись о найденном мусоре.
func (cfg *JunkConfig) AddRecord(path, categoryID string, size int64, programHint string) {
	// Проверяем дубликаты
	for i, r := range cfg.Records {
		if r.Path == path && r.CategoryID == categoryID {
			// Обновляем
			cfg.Records[i].Size = size
			cfg.Records[i].ProgramHint = programHint
			return
		}
	}
	cfg.Records = append(cfg.Records, JunkRecord{
		Path:        path,
		CategoryID:  categoryID,
		Size:        size,
		ProgramHint: programHint,
	})
	cfg.Stats[categoryID] += size
}

// MarkDeleted помечает запись как удалённую.
func (cfg *JunkConfig) MarkDeleted(path, categoryID string) {
	for i, r := range cfg.Records {
		if r.Path == path && r.CategoryID == categoryID && r.DeletedAt.IsZero() {
			cfg.Records[i].DeletedAt = time.Now()
			return
		}
	}
}

// GetNonDeletedRecords возвращает все ещё существующие записи.
func (cfg *JunkConfig) GetNonDeletedRecords() []JunkRecord {
	var out []JunkRecord
	for _, r := range cfg.Records {
		if r.DeletedAt.IsZero() {
			out = append(out, r)
		}
	}
	return out
}

// RemoveRecord удаляет запись из конфига.
func (cfg *JunkConfig) RemoveRecord(path, categoryID string) {
	for i, r := range cfg.Records {
		if r.Path == path && r.CategoryID == categoryID {
			cfg.Records = append(cfg.Records[:i], cfg.Records[i+1:]...)
			if cfg.Stats[r.CategoryID] > 0 {
				cfg.Stats[r.CategoryID] -= r.Size
			}
			return
		}
	}
}

// GetRecordsByCategory возвращает записи по категории.
func (cfg *JunkConfig) GetRecordsByCategory(categoryID string) []JunkRecord {
	var out []JunkRecord
	for _, r := range cfg.Records {
		if r.CategoryID == categoryID {
			out = append(out, r)
		}
	}
	return out
}

// GetRecordsByProgram возвращает записи по подсказке программы.
func (cfg *JunkConfig) GetRecordsByProgram(programHint string) []JunkRecord {
	var out []JunkRecord
	for _, r := range cfg.Records {
		if r.ProgramHint != "" && programHint != "" &&
			(r.ProgramHint == programHint || containsString(r.ProgramHint, programHint)) {
			out = append(out, r)
		}
	}
	return out
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
