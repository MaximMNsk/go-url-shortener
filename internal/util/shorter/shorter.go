package shorter

import "fmt"

// GetShortURL - возвращает сокращенный УРЛ из адреса и идентификатора УРЛ.
func GetShortURL(hostPort, linkID string) string {
	return fmt.Sprintf("%s/%s", hostPort, linkID)
}
