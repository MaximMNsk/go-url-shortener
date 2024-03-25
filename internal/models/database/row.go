package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"time"
)

type ErrorDB struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

func (e *ErrorDB) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

const layer = `DB`

type DBStorage struct {
	Ctx            context.Context
	Link           string `json:"original_url"`
	ShortLink      string `json:"short_url"`
	ID             string `json:"correlation_id"`
	DeletedFlag    bool   `json:"is_deleted"`
	ToDeleteCh     chan DeleteItem
	ConnectionPool *pgxpool.Pool
	Cfg            confModule.OuterConfig
}

func (jsonData *DBStorage) Init(link, shortLink, id string, isDeleted bool, ctx context.Context, cfg confModule.OuterConfig) {
	jsonData.Ctx = ctx
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.DeletedFlag = isDeleted
	jsonData.Cfg = cfg
}

func (jsonData *DBStorage) Destroy() {
	db.Close(jsonData.ConnectionPool)
}

const insertLinkRow = `
insert into public.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const insertLinkRowBatch = `

insert into public.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const selectRow = `
select uid, original_url, short_url, is_deleted from public.short_links where (uid = $1 or original_url = $2)`

const selectRowByUser = `
select uid, original_url, short_url from public.short_links where (uid = $1 or original_url = $2) and user_id = $3`

const selectAllRows = `
select original_url, short_url from public.short_links where user_id = $1`

const updateRow = `
update public.short_links set is_deleted = true where uid = $1 and user_id = $2`

const updateRowNoUser = `
update public.short_links set is_deleted = true where uid = $1`

func PrepareDB(dsn string) error {

	prepareErr := ErrorDB{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `PrepareDB`,
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

func (jsonData *DBStorage) Ping() (bool, error) {

	err := jsonData.ConnectionPool.Ping(jsonData.Ctx)
	if err != nil {
		pingErr := fmt.Errorf(`%w`, &ErrorDB{
			layer:          layer,
			parentFuncName: `-`,
			funcName:       `Ping`,
			message:        err.Error(),
		})
		return false, pingErr
	}
	return true, nil
}

func (jsonData *DBStorage) Get() (string, bool, error) {
	getErr := ErrorDB{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `Get`,
		message:        `Error occurred`,
	}

	row, err := getData(*jsonData)
	if err != nil {
		return row.Link, row.DeletedFlag, fmt.Errorf(getErr.Error()+`%w`, err)
	}
	return row.Link, row.DeletedFlag, nil
}

func getData(data DBStorage) (DBStorage, error) {

	var selected DBStorage
	connection := data.ConnectionPool

	getDataErr := ErrorDB{
		layer:          layer,
		parentFuncName: `Get`,
		funcName:       `getData`,
	}

	userID := data.Ctx.Value(cookie.UserNum(`UserID`))

	acquire, err := connection.Acquire(data.Ctx)
	if err != nil {
		getDataErr.message = err.Error()
		return selected, &getDataErr
	}
	defer acquire.Release()

	if connection == nil {
		connErr := errors.New("connection to DB not found")
		getDataErr.message = connErr.Error()
		return selected, &getDataErr
	}
	query := selectRow
	row := acquire.QueryRow(data.Ctx, query, data.ID, data.Link)

	err = row.Scan(&selected.ID, &selected.Link, &selected.ShortLink, &selected.DeletedFlag)
	if err != nil {
		getDataErr.message = fmt.Sprintf(`Error: %v, ID: %s, Link: %s, UserID: %s`,
			err.Error(), data.ID, data.Link, userID)
		return selected, &getDataErr
	}

	return selected, nil
}

func (jsonData *DBStorage) Set() error {

	errSet := ErrorDB{
		layer:          layer,
		funcName:       `Set`,
		parentFuncName: `-`,
	}

	err := saveData(*jsonData)

	if err != nil {
		errSet.message = `cannot set data to database`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}
	return nil
}

func saveData(data DBStorage) error {

	errSave := ErrorDB{
		layer:          layer,
		funcName:       `saveData`,
		parentFuncName: `Set`,
	}

	ctx := data.Ctx
	connection := data.ConnectionPool
	if connection == nil {
		errSave.message = `connection to DB not found`
		return &errSave
	}

	acquire, err := connection.Acquire(ctx)
	if err != nil {
		errSave.message = `cant acquire connection`
		return fmt.Errorf(errSave.Error()+`: %w`, err)
	}
	defer acquire.Release()

	userID := `0`
	reqUserID := ctx.Value(cookie.UserNum(`UserID`))
	if reqUserID != nil {
		userID = reqUserID.(string)
	}

	_, err = acquire.Exec(ctx, insertLinkRow, data.Link, data.ShortLink, data.ID, userID)

	if err != nil {
		errSave.message = `cannot insert row`
		dbErr := fmt.Errorf(errSave.Error()+`: %w`, err)
		return fmt.Errorf(dbErr.Error()+`: %w`, err)
	}

	return nil
}

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func (jsonData *DBStorage) BatchSet() ([]byte, error) {

	var savingData []DBStorage
	var outputData []outputBatch

	errBatchSet := ErrorDB{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(jsonData.Cfg.Final.ShortURLAddr, v.ID)

		savingData[i].ID = v.ID
		savingData[i].ShortLink = shortLink
		savingData[i].Link = v.Link

		outputData = append(outputData, outputBatch{ShortURL: shortLink, CorrelationID: v.ID})
	}

	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`))

	if jsonData.ConnectionPool == nil {
		errBatchSet.message = "connection to DB not found"
		return nil, &errBatchSet
	}

	acquire, err := jsonData.ConnectionPool.Acquire(jsonData.Ctx)
	if err != nil {
		errBatchSet.message = "cannot acquire connection"
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}
	defer acquire.Release()

	var batch pgx.Batch
	for _, v := range savingData {
		batch.Queue(insertLinkRowBatch, v.Link, v.ShortLink, v.ID, userID.(string))
	}
	br := acquire.SendBatch(jsonData.Ctx, &batch)
	defer br.Close()
	_, errPg := br.Exec()

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

type JSONCutted struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
}

func (jsonData *DBStorage) HandleUserUrls() ([]byte, error) {
	var batchResp []JSONCutted

	errHandleUserUrls := ErrorDB{
		layer:          layer,
		funcName:       `HandleUserUrls`,
		parentFuncName: `-`,
	}

	if jsonData.ConnectionPool == nil {
		errHandleUserUrls.message = "connection to DB not found"
		return nil, &errHandleUserUrls
	}

	acquire, err := jsonData.ConnectionPool.Acquire(jsonData.Ctx)
	if err != nil {
		errHandleUserUrls.message = "cannot acquire connection"
		return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
	}
	defer acquire.Release()

	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`))

	rows, err := acquire.Query(jsonData.Ctx, selectAllRows, strconv.Itoa(userID.(int)))
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

type DeleteItem struct {
	URLs   string
	UserID int
}

var toDeleteCh chan DeleteItem

func (jsonData *DBStorage) HandleUserUrlsDelete() {
	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`)).(int)

	inputData := DeleteItem{
		URLs:   jsonData.Link,
		UserID: userID,
	}

	go func() {
		toDeleteCh <- inputData
	}()
}

func (jsonData *DBStorage) AsyncSaver() {
	toDeleteCh = make(chan DeleteItem)
	defer close(toDeleteCh)

	errHandleUserUrlsDelete := ErrorDB{
		layer:          layer,
		funcName:       `AsyncSaver`,
		parentFuncName: `-`,
	}

	for {
		if toDeleteCh == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		select {
		case data, ok := <-toDeleteCh:
			if !ok {
				errHandleUserUrlsDelete.message = `channel reading error`
				logger.PrintLog(logger.WARN, errHandleUserUrlsDelete.Error())
				continue
			}
			err := batchUpdate(data.URLs, jsonData.ConnectionPool)
			if err != nil {
				errHandleUserUrlsDelete.message = `update error`
				logger.PrintLog(logger.WARN, errHandleUserUrlsDelete.Error()+` `+err.Error())
				continue
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func ExplodeURLs(data string) ([]string, error) {

	errExplodeURLs := ErrorDB{
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

func batchUpdate(links string, pool *pgxpool.Pool) error {

	errBatchUpdate := ErrorDB{
		layer:          layer,
		funcName:       `errBatchUpdate`,
		parentFuncName: `AsyncSaver`,
	}

	data, err := ExplodeURLs(links)
	if err != nil {
		errBatchUpdate.message = `explode error`
		return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	}

	ctx := context.Background()
	if pool == nil {
		errBatchUpdate.message = `connection to DB not found`
		return &errBatchUpdate
	}

	acquire, err := pool.Acquire(ctx)
	if err != nil {
		errBatchUpdate.message = `acquire error`
		return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	}
	defer acquire.Release()

	var batch pgx.Batch
	for _, uid := range data {
		batch.Queue(updateRowNoUser, uid)
	}

	br := acquire.SendBatch(ctx, &batch)
	defer br.Close()
	_, err = br.Exec()

	if err != nil {
		errBatchUpdate.message = `batch update error`
		return fmt.Errorf(errBatchUpdate.Error()+`: %w`, err)
	}

	return err
}
