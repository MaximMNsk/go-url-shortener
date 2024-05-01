package memorystorage

// StorageItem - единица хранения УРЛ в памяти.
type StorageItem struct {
	Link        string
	ShortLink   string
	ID          string
	DeletedFlag bool
}

// Storage - хранилище в памяти.
type Storage struct {
	data []StorageItem
}

// Init - создает пустое хранилище.
func (s *Storage) Init() {
	s.data = make([]StorageItem, 0)
}

// Set - записывает единицу хранения в память.
func (s *Storage) Set(data StorageItem) {
	s.data = append(s.data, data)
}

// Get - получает из памяти все единицы хранения.
func (s *Storage) Get() []StorageItem {
	return s.data
}
