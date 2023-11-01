package database

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"sync"
)

type DBStorage struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
	ID        string `json:"correlation_id"`
	Ctx       context.Context
}

func (jsonData *DBStorage) Init(link, shortLink, id string, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
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
		user_id text
	)`

const createIndexQuery = `
CREATE UNIQUE INDEX IF NOT EXISTS unique_original_url
ON shortener.short_links(original_url)`

const insertLinkRow = `
insert into shortener.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const insertLinkRowBatch = `

insert into shortener.short_links (original_url, short_url, uid, user_id) values ($1, $2, $3, $4)`

const selectRow = `
select uid, original_url, short_url from shortener.short_links where (uid = $1 or original_url = $2)`

const selectRowByUser = `
select uid, original_url, short_url from shortener.short_links where (uid = $1 or original_url = $2) and user_id = $3`

const selectAllRows = `
select original_url, short_url from shortener.short_links where user_id = $1`

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

func (jsonData *DBStorage) Get() (string, error) {

	logger.PrintLog(logger.INFO, "Get from database")
	row, err := getData(*jsonData)
	return row.Link, err

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
	//userID := ctx.Value(`UserID`)
	//query := selectRowByUser
	//row := connection.QueryRow(ctx, query, data.ID, data.Link, userID)
	//if userID == `` || userID == nil {
	query := selectRow
	row := connection.QueryRow(ctx, query, data.ID, data.Link)
	//}

	err := row.Scan(&selected.ID, &selected.Link, &selected.ShortLink)
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

	userID := ctx.Value(`UserID`)

	_, err := connection.Exec(ctx, insertLinkRow, data.Link, data.ShortLink, data.ID, userID)
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

	userID := jsonData.Ctx.Value(`UserID`)

	///////// Current logic
	connection := db.GetDB()
	if connection == nil {
		return nil, errors.New("connection to DB not found")
	}

	batch := pgx.Batch{}
	for _, v := range savingData {
		batch.Queue(insertLinkRowBatch, v.Link, v.ShortLink, v.ID, userID)
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

	userID := jsonData.Ctx.Value(`UserID`)

	rows, err := connection.Query(jsonData.Ctx, selectAllRows, userID)
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
