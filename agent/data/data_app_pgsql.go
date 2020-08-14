package data

import (
	"clearview/common"
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
)

func GetDataAppPgsql(client *http.Client, config *common.Config, data *common.Data) error {

	if !data.HasProcess("postgres") {
		return nil
	}

	if !config.GetBoolean("pgsql_enabled", false) {
		return nil
	}

	data.AppPgsql = &common.DataAppPgsql{}

	user := config.Get("pgsql_username")
	pass := config.Get("pgsql_password")

	if len(user) <= 0 || len(pass) <= 0 {
		fmt.Printf("Please set PgSQL username and password in /etc/clearview.conf or disable with pgsql_enabled: no\n")
		return nil
	}

	connStr := fmt.Sprintf("user=%s password=%s", user, pass)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	err = db.QueryRow("SELECT version()").Scan(&data.AppPgsql.Version)
	if err != nil {
		return err
	}

	// https://www.postgresql.org/docs/current/monitoring-stats.html#PG-STAT-ALL-TABLES-VIEW

	rows, err := db.Query(`SELECT seq_scan, idx_scan,
			seq_tup_read, idx_tup_fetch,
			n_tup_ins, n_tup_upd, n_tup_del
  			FROM pg_stat_all_tables`)
	if err != nil {
		return err
	}

	for rows.Next() {
		table := common.DataAppPgsql{}

		err = rows.Scan(&table.SeqScanCumulative, &table.IdxScanCumulative,
			&table.RowsSeqFetchCumulative, &table.RowsIdxFetchCumulative,
			&table.RowsInsertCumulative, &table.RowsUpdateCumulative, &table.RowsDeleteCumulative)
		if err != nil {
			return err
		}

		data.AppPgsql.SeqScanCumulative = table.SeqScanCumulative
		data.AppPgsql.IdxScanCumulative = table.IdxScanCumulative

		data.AppPgsql.RowsSeqFetchCumulative = table.RowsSeqFetchCumulative
		data.AppPgsql.RowsIdxFetchCumulative = table.RowsIdxFetchCumulative

		data.AppPgsql.RowsSelectCumulative = table.RowsSeqFetchCumulative + table.RowsIdxFetchCumulative
		data.AppPgsql.RowsInsertCumulative = table.RowsInsertCumulative
		data.AppPgsql.RowsUpdateCumulative = table.RowsUpdateCumulative
		data.AppPgsql.RowsDeleteCumulative = table.RowsDeleteCumulative
	}
	return nil
}
