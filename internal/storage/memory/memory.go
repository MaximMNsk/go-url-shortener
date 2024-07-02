// Package memorystorage - определяет хранилище в памяти
package memorystorage

// StorageItem - единица хранения УРЛ в памяти.
type StorageItem struct {
	Link        string `json:"original_url"`
	ShortLink   string
	ID          string `json:"correlation_id"`
	DeletedFlag bool
}

// Storage - хранилище в памяти.
type Storage struct {
	data    []StorageItem
	enabled bool
}

// Init - создает пустое хранилище.
func (s *Storage) Init() {
	s.data = make([]StorageItem, 0)
	s.enabled = true
}

// Set - записывает единицу хранения в память.
func (s *Storage) Set(data StorageItem) {
	s.data = append(s.data, data)
}

// Get - получает из памяти все единицы хранения.
func (s *Storage) Get() []StorageItem {
	return s.data
}

// Clear - делает пустым хранилище.
func (s *Storage) Clear() {
	s.data = make([]StorageItem, 0)
}

// Enabled - возвращает статус хранилища
func (s *Storage) Enabled() bool {
	return s.enabled
}
