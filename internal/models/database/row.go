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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"math"
	"strconv"
	"sync"
)

type DBStorage struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
	Ctx         context.Context
}

func (jsonData *DBStorage) Init(link, shortLink, id string, isDeleted bool, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
	jsonData.DeletedFlag = isDeleted
}

const createSchemaQuery = `
CREATE SCHEMA IF NOT EXISTS shortener 
AUTHORIZATION postgres`

const createTableQuery = `
CREATE TABLE IF NOT EXISTS shortener.short_links 
	(
	    id serial primary key,
	    original_url text,
	    short_url text,
	    uid text,
		user_id text,
		is_deleted bool
	)`

const createIndexQuery = `
CREATE UNIQUE INDEX IF NOT EXISTS unique_original_url
ON shortener.short_links(original_url)`

const insertLinkRow = `
insert into shortener.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const insertLinkRowBatch = `

insert into shortener.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const selectRow = `
select uid, original_url, short_url, is_deleted from shortener.short_links where (uid = $1 or original_url = $2)`

const selectRowByUser = `
select uid, original_url, short_url from shortener.short_links where (uid = $1 or original_url = $2) and user_id = $3`

const selectAllRows = `
select original_url, short_url from shortener.short_links where user_id = $1`

const updateRow = `
update shortener.short_links set is_deleted = true where uid = $1 and user_id = $2`

func PrepareDB(connect *pgx.Conn) {
	_, err := connect.Exec(db.GetCtx(), createSchemaQuery)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't create schema: "+err.Error())
		return
	}

	_, err = connect.Exec(db.GetCtx(), createTableQuery)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't create table: "+err.Error())
		return
	}

	_, err = connect.Exec(db.GetCtx(), createIndexQuery)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't create index: "+err.Error())
		return
	}
}

func (jsonData *DBStorage) Get() (string, bool, error) {

	logger.PrintLog(logger.INFO, "Get from database")
	row, err := getData(*jsonData)
	return row.Link, row.DeletedFlag, err

}

func getData(data DBStorage) (DBStorage, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	ctx := data.Ctx
	selected := DBStorage{}
	connection := db.GetDB()
	if connection == nil {
		return selected, errors.New("connection to DB not found")
	}
	//userID := ctx.Value(cookie.UserNum(`UserID`))
	//query := selectRowByUser
	//row := connection.QueryRow(ctx, query, data.ID, data.Link, strconv.Itoa(userID.(int)))
	//if userID == `` || userID == nil {
	query := selectRow
	row := connection.QueryRow(ctx, query, data.ID, data.Link)
	//}

	err := row.Scan(&selected.ID, &selected.Link, &selected.ShortLink, &selected.DeletedFlag)
	if err != nil {
		logger.PrintLog(logger.WARN, "Select attention: "+err.Error())
	}
	return selected, nil
}

func (jsonData *DBStorage) Set() error {

	logger.PrintLog(logger.INFO, "Set to database")

	var err error
	*jsonData, err = saveData(*jsonData)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Can't save. Error: "+err.Error())
	}
	return err
}

func saveData(data DBStorage) (DBStorage, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	ctx := data.Ctx
	connection := db.GetDB()
	if connection == nil {
		return DBStorage{}, errors.New("connection to DB not found")
	}

	userID := ctx.Value(cookie.UserNum(`UserID`))

	_, err := connection.Exec(ctx, insertLinkRow, data.Link, data.ShortLink, data.ID, userID.(string))
	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)

	if pgErr != nil {
		logger.PrintLog(logger.WARN, "Insert attention: "+err.Error())
		return data, pgErr
	}
	return data, nil
}

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func (jsonData *DBStorage) BatchSet() ([]byte, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	var savingData []DBStorage
	var outputData []outputBatch

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		return nil, err
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, v.ID)

		savingData[i].ID = v.ID
		savingData[i].ShortLink = shortLink
		savingData[i].Link = v.Link

		outputData = append(outputData, outputBatch{ShortURL: shortLink, CorrelationID: v.ID})
	}

	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`))

	///////// Current logic
	connection := db.GetDB()
	if connection == nil {
		return nil, errors.New("connection to DB not found")
	}

	batch := pgx.Batch{}
	for _, v := range savingData {
		batch.Queue(insertLinkRowBatch, v.Link, v.ShortLink, v.ID, userID.(string))
	}
	br := connection.SendBatch(jsonData.Ctx, &batch)
	defer br.Close()
	_, err = br.Exec()

	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)
	//////// End logic

	JSONResp, err := json.Marshal(outputData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
		return nil, err
	}
	if pgErr != nil {
		return JSONResp, pgErr
	}

	return JSONResp, nil
}

type JSONCutted struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
}

func (jsonData *DBStorage) HandleUserUrls() ([]byte, error) {
	var batchResp []JSONCutted
	connection := db.GetDB()
	if connection == nil {
		return nil, errors.New("connection to DB not found")
	}

	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`))

	rows, err := connection.Query(jsonData.Ctx, selectAllRows, strconv.Itoa(userID.(int)))
	if err != nil {
		logger.PrintLog(logger.WARN, "Select attention: "+err.Error())
	}
	for rows.Next() {
		var selected JSONCutted
		_ = rows.Scan(&selected.Link, &selected.ShortLink)
		batchResp = append(batchResp, selected)
	}
	if len(batchResp) > 0 {
		JSONResp, err := json.Marshal(batchResp)
		return JSONResp, err
	}

	return nil, nil
}

type input struct {
	URLs   string
	UserID int
}

func (jsonData *DBStorage) HandleUserUrlsDelete() error {
	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`)).(int)
	doneCh := make(chan struct{})
	defer close(doneCh)
	maxWorkers := 2

	inputData := input{
		URLs:   jsonData.Link,
		UserID: userID,
	}

	update(doneCh, inputData, maxWorkers, jsonData.Ctx)

	return nil
}

func update(doneCh chan struct{}, data input, countWorkers int, ctx context.Context) {
	var wg sync.WaitGroup

	explodedData, err := explodeURLs(data.URLs)
	if err != nil {
		logger.PrintLog(logger.FATAL, err.Error())
		return
	}
	inputLen := len(explodedData)
	partLen := int(math.Ceil(float64(inputLen) / float64(countWorkers)))

	for i := 0; i < countWorkers; i++ {
		wg.Add(1)
		start := i * partLen
		end := (i + 1) * partLen
		if i+1 == countWorkers {
			end = inputLen
		}
		chunk := explodedData[start:end]

		go func() {
			defer wg.Done()

			select {
			case <-doneCh:
				return
			default:
				err := batchUpdate(chunk, data.UserID, ctx)
				if err != nil {
					logger.PrintLog(logger.ERROR, err.Error())
				}
			}
		}()
	}

	go func() {
		wg.Wait()
	}()
}

func explodeURLs(data string) ([]string, error) {
	var out []string
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		return make([]string, 0), err
	}
	return out, nil
}

func batchUpdate(data []string, userID int, ctx context.Context) error {
	connection := db.GetDB()
	if connection == nil {
		return errors.New("connection to DB not found")
	}

	batch := pgx.Batch{}
	for i, uid := range data {
		fmt.Printf(`I: %d, Uid: %s, userID: %d`+"\n", i, uid, userID)
		batch.Queue(updateRow, uid, strconv.Itoa(userID))
	}
	br := connection.SendBatch(ctx, &batch)
	defer br.Close()
	_, err := br.Exec()

	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)

	return err
}
