// Package models стандартизирует работу с хранилищами
package models

import (
	"context"
)

// Storable - интерфейс для создания
// новых хранилищ.
//
//go:generate go run github.com/vektra/mockery/v2@v2.43.0 --name=Storable
type Storable interface {

	// Init - метод Инициализации хранилища.
	Init() error

	// Get - получение данных из хранилища, где:
	// первое значение - данные,
	// второе значение - отметка об удалении,
	// третье - ошибка.
	Get(ctx context.Context, shortLink string) (string, bool, error)

	// Set - сохранение данных структуры объекта.
	Set(ctx context.Context, originalLink string, shortLink string, hashLink string, userID int) error

	// Ping - проверка работоспособности хранилища.
	Ping(ctx context.Context) (bool, error)

	// BatchSet - сохранение пакета значений, переданного в структуре объекта.
	// Возвращает первым значением JSON объект в виде байт-кода,
	// вторым - ошибку если есть.
	BatchSet(ctx context.Context, data []byte, userID int) ([]byte, error)

	// HandleUserUrls - аналог BatchSet для конкретного пользователя.
	HandleUserUrls(ctx context.Context, userID int) ([]byte, error)

	// HandleUserUrlsDelete - удаляет шортлинки для пользователя,
	// переданные в структуре объекта. Передает данные в канал.
	HandleUserUrlsDelete(links string, userID int)

	// AsyncSaver - асинхронно сохраняет инфо, переданную в канал методом HandleUserUrlsDelete.
	// Работает как демон.
	AsyncSaver()

	// Destroy - останавливает работу хранилища.
	// Рекомендуется использовать для Graceful Shutdown
	Destroy() error
}
