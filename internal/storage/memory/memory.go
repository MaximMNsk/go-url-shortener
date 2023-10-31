package memorystorage

type StorageItem struct {
	Link      string
	ShortLink string
	ID        string
}

type Storage struct {
	data []StorageItem
}

func (s *Storage) Init() {
	//s.data = make(map[string]StorageItem)
}

func (s *Storage) Set(data StorageItem) {
	s.data = append(s.data, data)
}

func (s *Storage) Get() []StorageItem {
	return s.data
}
