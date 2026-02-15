package mysql

import (
	"fmt"
	"net"
	"strconv"

	drivermysql "github.com/go-sql-driver/mysql"
)

type connInfo struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Database string
}

func parseDSN(dsn string) (connInfo, error) {
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		return connInfo{}, fmt.Errorf("mysql cdc: parse dsn: %w", err)
	}
	if cfg.Addr == "" {
		return connInfo{}, fmt.Errorf("mysql cdc: dsn missing addr")
	}
	host, portStr, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		// Addr may be host-only.
		host = cfg.Addr
		portStr = "3306"
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return connInfo{}, fmt.Errorf("mysql cdc: invalid port %q: %w", portStr, err)
	}
	return connInfo{
		Host:     host,
		Port:     uint16(port),
		User:     cfg.User,
		Password: cfg.Passwd,
		Database: cfg.DBName,
	}, nil
}
