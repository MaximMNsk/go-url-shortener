package shorter

import "fmt"

func GetShortURL(hostPort, linkID string) string {
	return fmt.Sprintf("%s/%s", hostPort, linkID)
}
