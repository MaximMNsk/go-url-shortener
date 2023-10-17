package database

import (
	"context"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/jackc/pgx/v5"
)

type JSONData struct {
	Link      string
	ShortLink string
	ID        string
}

const createSchemaQuery = `CREATE SCHEMA IF NOT EXISTS shortener AUTHORIZATION postgres`
const createTableQuery = `
CREATE TABLE IF NOT EXISTS shortener.short_links 
	(
	    id serial primary key,
	    link text,
	    shortlink text,
	    uid varchar(10)
	)`

const insertLinkRow = `insert into shortener.short_links (link, shortlink, uid) values ($1, $2, $3)`

const selectRow = `select uid, link, shortlink from shortener.short_links where uid = $1 or link = $2`

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
}

func (jsonData *JSONData) Get() error {
	logger.PrintLog(logger.INFO, "Get from database")
	row, err := getData(*jsonData)
	jsonData.ID = row.ID
	jsonData.Link = row.Link
	jsonData.ShortLink = row.ShortLink
	return err
}

func getData(data JSONData) (JSONData, error) {
	ctx := context.Background()
	connection := db.GetDB()
	if connection == nil {
		return JSONData{}, errors.New("connection to DB not found")
	}
	selected := JSONData{}
	row := connection.QueryRow(ctx, selectRow, data.ID, data.Link)
	if row != nil {
		err := row.Scan(&selected.ID, &selected.Link, &selected.ShortLink)
		if err != nil {
			logger.PrintLog(logger.WARN, "Select attention: "+err.Error())
		}
		return selected, nil
	}
	return JSONData{}, nil
}

func (jsonData *JSONData) Set() error {
	logger.PrintLog(logger.INFO, "Set to database")
	selected, err := getData(*jsonData)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Error: "+err.Error())
		return err
	}

	if selected.ID == jsonData.ID {
		logger.PrintLog(logger.INFO, "Setter. Link row already exists")
		return nil
	}

	err = saveData(*jsonData)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Can't save. Error: "+err.Error())
	}
	return err
}

func saveData(data JSONData) error {
	ctx := context.Background()
	connection := db.GetDB()
	if connection == nil {
		return errors.New("connection to DB not found")
	}

	_, err := connection.Exec(ctx, insertLinkRow, data.Link, data.ShortLink, data.ID)
	if err != nil {
		return err
	}

	return nil
}
