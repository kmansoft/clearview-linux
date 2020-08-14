package data

import (
	"clearview/common"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

func GetDataAppMysql(client *http.Client, config *common.Config, data *common.Data) error {

	if !data.HasProcess("mysqld") && !data.HasProcess("mysqld.bin") {
		return nil
	}

	if !config.GetBoolean("mysql_enabled", false) {
		return nil
	}

	data.AppMysql = &common.DataAppMysql{
		Instant:    make(map[string]uint64),
		Cumulative: make(map[string]uint64),
	}

	user := config.Get("mysql_username")
	pass := config.Get("mysql_password")

	if len(user) <= 0 || len(pass) <= 0 {
		fmt.Printf("Please set MySQL username and password in /etc/clearview.conf or disable with mysql_enabled: no\n")
		return nil
	}

	mysqlConfig := mysql.Config{
		Addr:                    "localhost",
		User:                    user,
		Passwd:                  pass,
		AllowNativePasswords:    true,
		AllowCleartextPasswords: true,
		AllowOldPasswords:       true,
	}
	mysqlDsn := mysqlConfig.FormatDSN()

	db, err := sql.Open("mysql", mysqlDsn)
	if err != nil {
		return nil
	}

	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	// Get data variables
	rows1, err := db.Query(`SHOW /*!50002 GLOBAL */ STATUS  WHERE Variable_name IN (
		"Com_select", "Com_insert", "Com_update", "Com_delete",
		"Slow_queries",
		"Bytes_sent", "Bytes_received",
		"Connections", "Max_used_connections", "Aborted_Connects", "Aborted_Clients",
		"Qcache_queries_in_cache", "Qcache_hits", "Qcache_inserts", "Qcache_not_cached", "Qcache_lowmem_prunes")`)
	defer func() {
		if rows1 != nil {
			_ = rows1.Close()
		}
	}()

	if err == nil {
		for rows1.Next() {
			var key, value string
			err := rows1.Scan(&key, &value)
			if err == nil {
				v, _ := strconv.ParseUint(value, 10, 64)
				if key == "Max_used_connections" ||
					key == "Slow_queries" ||
					key == "Aborted_Connects" ||
					key == "Aborted_Clients" {
					data.AppMysql.Instant[key] = v
				} else {
					data.AppMysql.Cumulative[key] = v
				}
			}
		}
	} else {
		fmt.Printf("Error executing query for rows1: %s\n", err)
	}

	// Get version variable
	var version string

	rows2, err := db.Query(`SHOW /*!50002 GLOBAL */ VARIABLES LIKE "version"`)
	defer func() {
		if rows2 != nil {
			_ = rows2.Close()
		}
	}()

	if err == nil {
		if rows2.Next() {
			var key, value string
			err := rows2.Scan(&key, &value)
			if err == nil {
				version = value
			}
		}
	} else {
		fmt.Printf("Error executing query for rows2: %s\n", err)
	}

	// Server version
	if len(version) > 0 {
		data.AppMysql.Version = version
	}

	return nil
}
