package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ChatDB struct {
	path string
	db   *sql.DB
}

func (m *ChatDB) Open(path string) error {
	m.path = path

	tableInit := !fileExists(m.path)

	db, err := sql.Open("sqlite3", m.path)
	if err == nil {
		m.db = db
		if tableInit {
			return m.createInternal()
		} else {
			return nil
		}
	} else {
		m.db = nil
		return err
	}
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (m *ChatDB) createInternal() error {
	stmt := `CREATE TABLE CHAT(
				ID STRING PRIMARY KEY,
				CREATED DATETIME,
				ERRCOUNT INTEGER);`

	_, err := m.db.Exec(stmt)
	return err
}

func (m *ChatDB) Close() error {
	if m.db != nil {
		return m.db.Close()
	} else {
		return errors.New("DB not open.")
	}
}

func (m *ChatDB) AddChat(chatId string) error {
	if m.isExistChat(chatId) {
		return nil
	}

	stmt := `INSERT INTO CHAT VALUES(?, ?, 0)`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return err
	}
	defer pstmt.Close()
	_, err = pstmt.Exec(chatId, time.Now())
	return err
}

func (m *ChatDB) isExistChat(chatId string) bool {
	stmt := `SELECT ID FROM CHAT WHERE ID = ?`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return false
	}
	defer pstmt.Close()
	row, err := pstmt.Query(chatId)
	if err != nil {
		return false
	}
	defer row.Close()
	return row.Next()
}

func (m *ChatDB) DelChat(chatId string) error {
	stmt := `DELETE FROM CHAT WHERE ID = ?`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return err
	}
	defer pstmt.Close()
	_, err = pstmt.Exec(chatId)
	return err
}

func (m *ChatDB) GetChatList() ([]string, error) {
	stmt := `SELECT ID FROM CHAT`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return nil, err
	}
	defer pstmt.Close()
	row, err := pstmt.Query()
	if err != nil {
		return nil, err
	}
	defer row.Close()

	ids := make([]string, 0)
	for row.Next() {
		var id string
		rowE := row.Scan(&id)
		if rowE == nil {
			ids = append(ids, id)
		} else {
			return nil, rowE
		}
	}

	return ids, nil
}

func (m *ChatDB) GetErrorChatList(errCount int) ([]string, error) {
	stmt := `SELECT ID FROM CHAT WHERE ERRCOUNT > ?`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return nil, err
	}
	defer pstmt.Close()
	row, err := pstmt.Query(errCount)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	ids := make([]string, 0)
	for row.Next() {
		var id string
		rowE := row.Scan(&id)
		if rowE == nil {
			ids = append(ids, id)
		} else {
			return nil, rowE
		}
	}

	return ids, nil
}

func (m *ChatDB) GetErrorCount(chatId string) (int, error) {
	stmt := `SELECT ERRCOUNT FROM CHAT WHERE ID = ?`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return -1, err
	}
	defer pstmt.Close()
	row, err := pstmt.Query(chatId)
	if err != nil {
		return -1, err
	}
	defer row.Close()

	if row.Next() {
		var count int
		rowE := row.Scan(&count)
		if rowE == nil {
			return count, nil
		} else {
			return -1, rowE
		}
	} else {
		return -1, errors.New(fmt.Sprintf("Chat ID %s not found.", chatId))
	}
}

func (m *ChatDB) SetErrorCount(chatId string, count int) error {
	stmt := `UPDATE CHAT SET ERRCOUNT = ? WHERE ID = ?`
	pstmt, err := m.db.Prepare(stmt)
	if err != nil {
		return err
	}
	defer pstmt.Close()
	_, err = pstmt.Exec(count, chatId)
	return err
}
