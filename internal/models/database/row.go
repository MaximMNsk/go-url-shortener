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
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"sync"
	"time"
)

type DBStorage struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
	ToDeleteCh  chan DeleteItem
	Ctx         context.Context
}

func (jsonData *DBStorage) Init(link, shortLink, id string, isDeleted bool, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
	jsonData.DeletedFlag = isDeleted
	jsonData.ToDeleteCh = make(chan DeleteItem, 100)
}

const createSchemaQuery = `
CREATE SCHEMA IF NOT EXISTS shortener 
AUTHORIZATION postgres`

const createTableQuery = `
CREATE TABLE IF NOT EXISTS shortener.short_links 
	(
	    id serial PRIMARY KEY,
	    original_url TEXT,
	    short_url TEXT,
	    uid TEXT,
		user_id TEXT,
		is_deleted BOOLEAN DEFAULT FALSE
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

func PrepareDB(connect *pgxpool.Pool, ctx context.Context) {
	_, err := connect.Exec(ctx, createSchemaQuery)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't create schema: "+err.Error())
		return
	}

	_, err = connect.Exec(ctx, createTableQuery)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't create table: "+err.Error())
		return
	}

	_, err = connect.Exec(ctx, createIndexQuery)
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
	var selected DBStorage
	connection := db.GetDB()

	acquire, err := connection.Acquire(ctx)
	if err != nil {
		return selected, err
	}
	defer acquire.Release()

	if connection == nil {
		return selected, errors.New("connection to DB not found")
	}
	query := selectRow
	row := acquire.QueryRow(ctx, query, data.ID, data.Link)

	err = row.Scan(&selected.ID, &selected.Link, &selected.ShortLink, &selected.DeletedFlag)
	if err != nil {
		logger.PrintLog(logger.WARN, "Select attention: "+err.Error())
		errData := fmt.Sprintf("ID: %s, Link: %s, ShortLink: %s, DeletedFlag: %v", selected.ID, selected.Link, selected.ShortLink, selected.DeletedFlag)
		logger.PrintLog(logger.WARN, "Select content: "+errData)
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

	acquire, err := connection.Acquire(ctx)
	if err != nil {
		return DBStorage{}, err
	}
	defer acquire.Release()

	userID := ctx.Value(cookie.UserNum(`UserID`))

	_, err = acquire.Exec(ctx, insertLinkRow, data.Link, data.ShortLink, data.ID, userID.(string))
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

	acquire, err := connection.Acquire(jsonData.Ctx)
	if err != nil {
		return nil, err
	}
	defer acquire.Release()

	var batch pgx.Batch
	for _, v := range savingData {
		batch.Queue(insertLinkRowBatch, v.Link, v.ShortLink, v.ID, userID.(string))
	}
	br := acquire.SendBatch(jsonData.Ctx, &batch)
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

	acquire, err := connection.Acquire(jsonData.Ctx)
	if err != nil {
		return nil, err
	}
	defer acquire.Release()

	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`))

	rows, err := acquire.Query(jsonData.Ctx, selectAllRows, strconv.Itoa(userID.(int)))
	if err != nil {
		logger.PrintLog(logger.WARN, "Select attention: "+err.Error())
	}
	for rows.Next() {
		var selected JSONCutted
		err = rows.Scan(&selected.Link, &selected.ShortLink)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Sort select result error: "+err.Error())
		}
		batchResp = append(batchResp, selected)
	}
	if len(batchResp) > 0 {
		JSONResp, err := json.Marshal(batchResp)
		return JSONResp, err
	}

	return nil, nil
}

type DeleteItem struct {
	URLs   string
	UserID int
}

func (jsonData *DBStorage) HandleUserUrlsDelete() {
	userID := jsonData.Ctx.Value(cookie.UserNum(`UserID`)).(int)
	doneCh := make(chan struct{})
	defer close(doneCh)

	inputData := DeleteItem{
		URLs:   jsonData.Link,
		UserID: userID,
	}

	go func() {
		jsonData.ToDeleteCh <- inputData
		fmt.Println(`Data sent`)
	}()
}

func (jsonData *DBStorage) AsyncSaver() {
	for {
		//fmt.Println(jsonData.ToDeleteChs)
		if jsonData.ToDeleteCh == nil {
			fmt.Println(`Nil channel`)
			time.Sleep(1 * time.Second)
			continue
		}
		select {
		case data, ok := <-jsonData.ToDeleteCh:
			if !ok {
				fmt.Println(`!ok`)
				continue
			}
			fmt.Println(`Save data:`)
			fmt.Println(data)
			err := batchUpdate(data.URLs, data.UserID)
			if err != nil {
				fmt.Println(err)
				continue
			}
		default:
			fmt.Println(`Listen channel`)
		}
		time.Sleep(1 * time.Second)
	}
}

//func createCh(data DeleteItem) chan DeleteItem {
//	inputCh := make(chan DeleteItem, 1)
//	inputCh <- data // Не работает, если не указана емкость канала!!!
//	close(inputCh)
//	return inputCh
//}
//
//func (jsonData *DBStorage) AppendCh(data DeleteItem, wg *sync.WaitGroup) {
//	inputCh := createCh(data)
//	jsonData.ToDeleteChs = append(jsonData.ToDeleteChs, inputCh)
//}

//func (jsonData *DBStorage) AsyncSaverOld() {
//	logger.PrintLog(logger.INFO, "Starting async saver")
//	for {
//		toDeleteChsCount := len(jsonData.ToDeleteChs)
//		fmt.Println(`Count channels in slice: ` + strconv.Itoa(toDeleteChsCount))
//		if toDeleteChsCount == 0 {
//			time.Sleep(1 * time.Second)
//			fmt.Println(`Empty slice. Retry...`)
//			continue
//		}
//		fmt.Println(`Not empty slice:`)
//		fmt.Println(jsonData.ToDeleteChs)
//
//		for i, ch := range jsonData.ToDeleteChs {
//		Loop:
//			for {
//				fmt.Println(i)
//				startLogWord := `Continue `
//				if i == 0 {
//					startLogWord = `Start `
//				}
//				fmt.Println(startLogWord + `iter not empty channel`)
//				select {
//				case inp, ok := <-ch:
//					fmt.Println(inp)
//					fmt.Println(ok)
//					fmt.Println(jsonData.ToDeleteChs)
//					emptyItem := DeleteItem{}
//					if !ok || inp == emptyItem {
//						fmt.Println(`!ok or empty Item. Exit iter channel`)
//						break Loop
//					}
//					err := batchUpdate(inp.URLs, inp.UserID)
//					jsonData.ToDeleteChs = jsonData.ToDeleteChs[i+1:]
//					if err != nil {
//						fmt.Println(err)
//						continue
//					}
//				}
//				time.Sleep(1 * time.Second)
//			} // Loop
//		}
//
//	}
//}

func explodeURLs(data string) ([]string, error) {
	var out []string
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		return make([]string, 0), err
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

func batchUpdate(links string, userID int) error {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	data, err := explodeURLs(links)
	if err != nil {
		fmt.Println(err)
	}

	connection := db.GetDB()
	ctx := context.Background()
	if connection == nil {
		return errors.New("connection to DB not found")
	}

	acquire, err := connection.Acquire(ctx)
	if err != nil {
		return err
	}
	defer acquire.Release()

	var batch pgx.Batch
	for _, uid := range data {
		//fmt.Printf(`I: %d, Uid: %s, userID: %d`+"\n", i, uid, userID)
		batch.Queue(updateRow, uid, strconv.Itoa(userID))
	}

	br := acquire.SendBatch(ctx, &batch)
	defer br.Close()
	_, err = br.Exec()

	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)

	return err
}
