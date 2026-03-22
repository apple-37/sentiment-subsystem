package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // 导入驱动
	"github.com/go-redis/redis/v8"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// 配置连接池
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLStore{db: db}, nil
}

// ArchiveDailyHotWords 批量将当日热词归档到 MySQL
func (m *MySQLStore) ArchiveDailyHotWords(country string, words[]redis.Z) error {
	if len(words) == 0 {
		return nil
	}

	// 使用事务批量插入
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	
	today := time.Now().Format("2006-01-02")
	stmt, err := tx.Prepare("INSERT INTO daily_hot_keywords (country, keyword, score, record_date) VALUES (?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, w := range words {
		_, err := stmt.Exec(country, fmt.Sprintf("%v", w.Member), w.Score, today)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (m *MySQLStore) Close() {
	if m.db != nil {
		m.db.Close()
	}
}