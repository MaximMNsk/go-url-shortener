// Package database - прикладной пакет для работы с БД.
package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

// DBError - определение ошибки слоя БД.
type DBError struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

// Error - заменяем стандартный вызов метода своим.
func (e *DBError) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

const layer = `DB`

// DBStorage - структура объекта, который создается при инициализации
// и используется для взаимодействия с хранилищем.
//
// Состоит из:
// ToDeleteCh - канал для массового удаления УРЛ.
// ConnectionPool - пул, из которого выбирается соединение для работы с хранилищем.
// Cfg - конфигурация, которая инициализируется при запуске.
type DBStorage struct {
	ToDeleteCh       chan DeleteItem
	AsyncSaverStatCh chan DBError
	ConnectionPool   db.Pool
	Cfg              confModule.OuterConfig
}

// Init - метод создает для каждого запроса объект.
func (dbs *DBStorage) Init() error {
	dbs.ToDeleteCh = make(chan DeleteItem)
	dbs.AsyncSaverStatCh = make(chan DBError)
	err := prepare(dbs.Cfg.Final.DB)
	return err
}

// Destroy - метод утилизирует объект для работы с хранилищем.
func (dbs *DBStorage) Destroy() error {
	if dbs.ToDeleteCh != nil {
		close(dbs.ToDeleteCh)
	}
	if dbs.AsyncSaverStatCh != nil {
		close(dbs.AsyncSaverStatCh)
	}
	dbs.ConnectionPool.Close()
	return nil
}

const connectError = `connection to DB not found`
const poolIsNilError = `connection pool is nil`

const insertLinkRow = `
insert into public.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const insertLinkRowBatch = `

insert into public.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const selectRow = `
select original_url, is_deleted from public.short_links where (uid = $1 or original_url = $1)`

const selectAllRows = `
select original_url, short_url from public.short_links where user_id = $1`

//const updateRow = `
//update public.short_links set is_deleted = true where uid = $1 and user_id = $2`

const updateRowNoUser = `
update public.short_links set is_deleted = true where uid = $1`

func prepare(dsn string) error {

	prepareErr := DBError{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `prepare`,
	}

	m, err := migrate.New(
		`file://internal/storage/db/migrations`,
		dsn)
	if err != nil {
		prepareErr.message = `initialization error`
		return fmt.Errorf(prepareErr.Error()+`: %w`, err)
	}
	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		prepareErr.message = `migrate error`
		return fmt.Errorf(prepareErr.Error()+`: %w`, err)
	}

	return nil
}

// Ping - прикладной метод для проверки работоспособности хранилища.
func (dbs *DBStorage) Ping(ctx context.Context) (bool, error) {

	if dbs.ConnectionPool == nil {
		return false, fmt.Errorf(`%w`, &DBError{
			layer:          layer,
			parentFuncName: `-`,
			funcName:       `Ping`,
			message:        poolIsNilError,
		})
	}

	err := dbs.ConnectionPool.Ping(ctx)
	if err != nil {
		pingErr := fmt.Errorf(`%w`, &DBError{
			layer:          layer,
			parentFuncName: `-`,
			funcName:       `Ping`,
			message:        err.Error(),
		})
		return false, pingErr
	}
	return true, nil
}

// Get - возвращает инфо о сохраненном и сокращенном УРЛ.
// Первый возвращаемый параметр - сокращенный УРЛ,
// второй - флаг присутствия,
// третий - ошибка выполнения.
func (dbs *DBStorage) Get(ctx context.Context, requestID string) (string, bool, error) {

	if dbs.ConnectionPool == nil {
		return ``, false, fmt.Errorf(`%w`, &DBError{
			layer:          layer,
			parentFuncName: `-`,
			funcName:       `Get`,
			message:        poolIsNilError,
		})
	}
	getErr := DBError{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `Get`,
		message:        `Error occurred`,
	}

	//acquire, err := dbs.ConnectionPool.Acquire(ctx)
	//if err != nil {
	//	getErr.message = err.Error()
	//	return ``, false, &getErr
	//}
	//defer acquire.Release()
	//
	//if acquire == nil {
	//	connErr := errors.New(connectError)
	//	getErr.message = connErr.Error()
	//	return ``, false, &getErr
	//}

	var URL string
	var isDeleted bool

	row := dbs.ConnectionPool.QueryRow(ctx, selectRow, requestID)

	err := row.Scan(&URL, &isDeleted)
	if err != nil {
		getErr.message = fmt.Sprintf(`Error: %v, ID: %s`,
			err, requestID)
		return ``, false, &getErr
	}

	return URL, isDeleted, nil

}

// Set - сохраняет и сокращает УРЛ.
// Возвращает статус работы в виде ошибки.
func (dbs *DBStorage) Set(ctx context.Context, originalLink string, shortLink string, hashLink string, userID int) error {

	if dbs.ConnectionPool == nil {
		return fmt.Errorf(`%w`, &DBError{
			layer:          layer,
			parentFuncName: `-`,
			funcName:       `Set`,
			message:        poolIsNilError,
		})
	}

	errSet := DBError{
		layer:          layer,
		funcName:       `Set`,
		parentFuncName: `-`,
	}

	//acquire, err := dbs.ConnectionPool.Acquire(ctx)
	//if err != nil {
	//	errSet.message = `cant acquire connection`
	//	return fmt.Errorf(errSet.Error()+`: %w`, err)
	//}
	//defer acquire.Release()

	_, err := dbs.ConnectionPool.Exec(ctx, insertLinkRow, originalLink, shortLink, hashLink, userID)

	if err != nil {
		errSet.message = `cannot insert row`
		dbErr := fmt.Errorf(errSet.Error()+`: %w`, err)
		return fmt.Errorf(dbErr.Error()+`: %w`, err)
	}

	return nil
}

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type inputBatch struct {
	CorrelationID string `json:"correlation_id"`
	OriginalLink  string `json:"original_url"`
	ShortLink     string
}

// BatchSet - сохраняет и сокращает УРЛ пакетно.
// Возвращает слайс сокращенных УРЛ в байт-формате, а так же результат выполнения.
func (dbs *DBStorage) BatchSet(ctx context.Context, data []byte, userID int) ([]byte, error) {

	errBatchSet := DBError{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	var savingData []inputBatch
	err := json.Unmarshal(data, &savingData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	outputData := make([]outputBatch, 0, len(savingData))
	for i, v := range savingData {
		shortLink := shorter.GetShortURL(dbs.Cfg.Final.ShortURLAddr, v.CorrelationID)

		savingData[i].CorrelationID = v.CorrelationID
		savingData[i].ShortLink = shortLink
		savingData[i].OriginalLink = v.OriginalLink

		outputData = append(outputData, outputBatch{ShortURL: shortLink, CorrelationID: v.CorrelationID})
	}

	if dbs.ConnectionPool == nil {
		errBatchSet.message = connectError
		return nil, &errBatchSet
	}

	//acquire, err := dbs.ConnectionPool.Acquire(ctx)
	//if err != nil {
	//	errBatchSet.message = "cannot acquire connection"
	//	return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	//}
	//defer acquire.Release()

	var batch pgx.Batch
	for _, v := range savingData {
		batch.Queue(insertLinkRowBatch, v.OriginalLink, v.ShortLink, v.CorrelationID, userID)
	}
	br := dbs.ConnectionPool.SendBatch(ctx, &batch)

	_, errPg := br.Exec()

	err = br.Close()
	if err != nil {
		errBatchSet.message = "cannot close batch"
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	JSONResp, err := json.Marshal(outputData)

	if errPg != nil {
		errBatchSet.message = "batch insert error"
		return JSONResp, fmt.Errorf(errBatchSet.Error()+`: %w`, errPg)
	}

	if err != nil {
		errBatchSet.message = "unmarshal error"
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	return JSONResp, nil
}

// JSONCutted - структура хранения входных/выходных данных для каждого УРЛ в пачке.
type JSONCutted struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
}

// HandleUserUrls - возвращает слайс УРЛ, сохраненных текущим пользователем.
// Так же возвращает результат обработки запроса.
func (dbs *DBStorage) HandleUserUrls(ctx context.Context, userID int) ([]byte, error) {
	var batchResp []JSONCutted

	errHandleUserUrls := DBError{
		layer:          layer,
		funcName:       `HandleUserUrls`,
		parentFuncName: `-`,
	}

	if dbs.ConnectionPool == nil {
		errHandleUserUrls.message = connectError
		return nil, &errHandleUserUrls
	}

	//acquire, err := dbs.ConnectionPool.Acquire(ctx)
	//if err != nil {
	//	errHandleUserUrls.message = "cannot acquire connection"
	//	return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
	//}
	//defer acquire.Release()

	rows, err := dbs.ConnectionPool.Query(ctx, selectAllRows, userID)
	if err != nil {
		errHandleUserUrls.message = "select error"
		return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
	}
	for rows.Next() {
		var selected JSONCutted
		err = rows.Scan(&selected.Link, &selected.ShortLink)
		if err != nil {
			errHandleUserUrls.message = "fetch error"
			return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
		}
		batchResp = append(batchResp, selected)
	}
	if len(batchResp) > 0 {
		JSONResp, err := json.Marshal(batchResp)
		if err != nil {
			errHandleUserUrls.message = "marshal error"
			return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
		}
		return JSONResp, nil
	}

	return nil, nil
}

// DeleteItem - структура хранения данных для удаления УРЛ из БД.
type DeleteItem struct {
	URLs   string
	UserID int
}

// HandleUserUrlsDelete - удаляет переданные УРЛ текущего пользователя.
// Отправляет данные в канал, из которого асинхронно вычитываются УРЛ и удаляются.
func (dbs *DBStorage) HandleUserUrlsDelete(links string, userID int) {

	inputData := DeleteItem{
		URLs:   links,
		UserID: userID,
	}

	go func() {
		dbs.ToDeleteCh <- inputData
	}()
}

// AsyncSaver - метод-демон, который работает асинхронно.
// Слушает канал, в который передаются УРЛ для удаления и обрабатывает их.
func (dbs *DBStorage) AsyncSaver() {

	errHandleUserUrlsDelete := DBError{
		layer:          layer,
		funcName:       `AsyncSaver`,
		parentFuncName: `-`,
	}

	if dbs.ConnectionPool == nil {
		errHandleUserUrlsDelete.message = connectError
		dbs.AsyncSaverStatCh <- errHandleUserUrlsDelete
	}

	for {
		if dbs.ToDeleteCh == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		select {
		case data, ok := <-dbs.ToDeleteCh:
			if !ok {
				errHandleUserUrlsDelete.message = `channel reading error`
				dbs.AsyncSaverStatCh <- errHandleUserUrlsDelete
				continue
			}
			err := dbs.BatchUpdate(context.Background(), data.URLs, data.UserID)
			if err != nil {
				errHandleUserUrlsDelete.message = `update error`
				dbs.AsyncSaverStatCh <- errHandleUserUrlsDelete
				continue
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ExplodeURLs - функция для парсинга json-строки.
func explodeURLs(data string) ([]string, error) {

	errExplodeURLs := DBError{
		layer:          layer,
		funcName:       `explodeURLs`,
		parentFuncName: `batchUpdate`,
	}

	var out []string
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		errExplodeURLs.message = `unmarshal error`
		return make([]string, 0), fmt.Errorf(errExplodeURLs.Error()+`: %w`, err)
	}
	var uniqueResult = make(map[string]bool)
	for _, v := range out {
		uniqueResult[v] = false
	}
	var result = make([]string, 0)
	for z := range uniqueResult {
		result = append(result, z)
	}
	return result, nil
}

// BatchUpdate - устанавливает флаг "удалено" для переданных УРЛ.
func (dbs *DBStorage) BatchUpdate(ctx context.Context, links string, _ int) error {

	errBatchUpdate := DBError{
		layer:          layer,
		funcName:       `errBatchUpdate`,
		parentFuncName: `AsyncSaver`,
	}

	data, err := explodeURLs(links)
	if err != nil {
		errBatchUpdate.message = `explode error`
		return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	}

	//acquire, err := dbs.ConnectionPool.Acquire(ctx)
	//if err != nil {
	//	errBatchUpdate.message = `acquire error`
	//	return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	//}
	//defer acquire.Release()

	var batch pgx.Batch
	for _, uid := range data {
		batch.Queue(updateRowNoUser, uid)
	}

	br := dbs.ConnectionPool.SendBatch(ctx, &batch)

	_, err = br.Exec()

	if err != nil {
		errBatchUpdate.message = `batch update error`
		return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	}

	err = br.Close()
	if err != nil {
		errBatchUpdate.message = "cannot close batch"
		return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	}

	return err
}
