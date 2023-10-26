package database

import (
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5"
	"strings"
)

type DBStorage struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
	ID        string `json:"correlation_id"`
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
	    uid text
	)`

const createIndexQuery = `
CREATE UNIQUE INDEX IF NOT EXISTS unique_original_url
ON shortener.short_links(original_url)`

const insertLinkRow = `
insert into shortener.short_links (original_url, short_url, uid) values ($1, $2, $3)`

const insertLinkRowBatch = `

insert into shortener.short_links (original_url, short_url, uid) values ($1, $2, $3)`

const selectRow = `
select uid, original_url, short_url from shortener.short_links where uid = $1 or original_url = $2`

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
	ctx := db.GetCtx()
	connection := db.GetDB()
	selected := DBStorage{}
	if connection == nil {
		return selected, errors.New("connection to DB not found")
	}
	row := connection.QueryRow(ctx, selectRow, data.ID, data.Link)
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
	ctx := db.GetCtx()
	connection := db.GetDB()
	if connection == nil {
		return DBStorage{}, errors.New("connection to DB not found")
	}

	_, err := connection.Exec(ctx, insertLinkRow, data.Link, data.ShortLink, data.ID)
	if err != nil {
		logger.PrintLog(logger.WARN, "Insert attention: "+err.Error())
		return data, err
	}
	return data, nil
}

func (jsonData *DBStorage) BatchSet() ([]byte, error) {

	var savingData []DBStorage

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		return nil, err
	}

	for i, v := range savingData {
		//linkID := sha1hash.Create(v.Link, 8)
		savingData[i].ID = v.ID
		savingData[i].ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, v.ID)
		savingData[i].Link = v.Link
	}

	///////// Current logic
	connection := db.GetDB()
	if connection == nil {
		return nil, errors.New("connection to DB not found")
	}

	batch := pgx.Batch{}
	for _, v := range savingData {
		batch.Queue(insertLinkRowBatch, v.Link, v.ShortLink, v.ID)
	}
	br := connection.SendBatch(db.GetCtx(), &batch)
	defer br.Close()
	_, err = br.Exec()
	if err != nil && !strings.Contains(err.Error(), `SQLSTATE 23505`) {
		return nil, err
	}
	//err = br.Close()
	//if err != nil {
	//	return nil, err
	//}
	//////// End logic

	JSONResp, err := json.Marshal(savingData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
	}

	return JSONResp, nil
}
