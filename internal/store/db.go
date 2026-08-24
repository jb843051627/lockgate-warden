package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// notFound 领域哨兵别名。
var notFound = model.ErrNotFound

// Store SQLite 落盘存储（单写连接池）。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）数据库文件并执行迁移。
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭底层连接。
func (s *Store) Close() error { return s.db.Close() }

// Transaction 在单个事务内执行 fn；返回 fn 的错误并回滚。
func (s *Store) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// requireAffected 断言写操作至少影响一行；否则返回 sentinel。
func requireAffected(res sql.Result, sentinel error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sentinel
	}
	return nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
