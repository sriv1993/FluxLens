package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-mysql-org/go-mysql/mysql"
)

func masterPosition(dsn string) (mysql.Position, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return mysql.Position{}, err
	}
	defer db.Close()
	var file string
	var pos uint32
	var doDB, ignoreDB, gtid sql.NullString
	err = db.QueryRow("SHOW MASTER STATUS").Scan(&file, &pos, &doDB, &ignoreDB, &gtid)
	if err != nil {
		return mysql.Position{}, fmt.Errorf("mysql cdc: show master status: %w", err)
	}
	return mysql.Position{Name: file, Pos: pos}, nil
}
