// Package models стандартизирует работу с хранилищами
package models

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
)

// Storable - интерфейс для создания
// новых хранилищ.
// go:generate go run github.com/vektra/mockery/v2@v2.43.0 --name=Storable
type Storable interface {

	// Init - метод Инициализации хранилища.
	Init(link, shortLink, id string, isDeleted bool, ctx context.Context, cfg config.OuterConfig) error

	// Get - получение данных из хранилища, где:
	// первое значение - данные,
	// второе значение - отметка об удалении,
	// третье - ошибка.
	Get() (string, bool, error)

	// Set - сохранение данных структуры объекта.
	Set() error

	// Ping - проверка работоспособности хранилища.
	Ping() (bool, error)

	// BatchSet - сохранение пакета значений, переданного в структуре объекта.
	// Возвращает первым значением JSON объект в виде байт-кода,
	// вторым - ошибку если есть.
	BatchSet() ([]byte, error)

	// HandleUserUrls - аналог BatchSet для конкретного пользователя.
	HandleUserUrls() ([]byte, error)

	// HandleUserUrlsDelete - удаляет шортлинки для пользователя,
	// переданные в структуре объекта. Передает данные в канал.
	HandleUserUrlsDelete()

	// AsyncSaver - асинхронно сохраняет инфо, переданную в канал методом HandleUserUrlsDelete.
	// Работает как демон.
	AsyncSaver()

	// Destroy - останавливает работу хранилища.
	// Рекомендуется использовать для Graceful Shutdown
	Destroy()
}
