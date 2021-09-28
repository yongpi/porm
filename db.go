package porm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/yongpi/putil/plog"
)

type MapperGetter interface {
	Mapper() *mapper
}

type QueryP interface {
	QueryP(dest interface{}, query string, args ...interface{}) error
	QueryContextP(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

type Query interface {
	MapperGetter
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type StmtQuery interface {
	MapperGetter
	QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error)
}

type DB struct {
	*sql.DB
	mapper *mapper
	Name   string
}

func (db *DB) Mapper() *mapper {
	return db.mapper
}

func (db *DB) QueryP(dest interface{}, query string, args ...interface{}) error {
	return QueryScan(context.Background(), db, dest, query, args...)
}

func (db *DB) QueryContextP(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return QueryScan(ctx, db, dest, query, args...)
}

func OpenDBName(driverName, dataSourceName, dbName string) (*DB, error) {
	sqlDB, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	db := &DB{DB: sqlDB, mapper: NewMapper(dbName), Name: dbName}
	return db, nil
}

func Open(driverName, dataSourceName string) (*DB, error) {
	return OpenDBName(driverName, dataSourceName, "db")
}

func (db *DB) ConnP(ctx context.Context) (*Conn, error) {
	sqlConn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	conn := &Conn{Conn: sqlConn, mapper: db.mapper}
	return conn, nil
}

func (db *DB) BeginP() (*Tx, error) {
	return db.BeginTxP(context.Background(), nil)
}

func (db *DB) BeginTxP(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	sqlTx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	tx := &Tx{Tx: sqlTx, mapper: db.mapper}
	return tx, nil
}

func (db *DB) PrepareContextP(ctx context.Context, query string) (*Stmt, error) {
	sqlStmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	stmt := &Stmt{Stmt: sqlStmt, mapper: db.mapper}
	return stmt, nil
}

func (db *DB) PrepareP(query string) (*Stmt, error) {
	return db.PrepareContextP(context.Background(), query)
}

type Conn struct {
	*sql.Conn
	mapper *mapper
}

func (c *Conn) Mapper() *mapper {
	return c.mapper
}

func (c *Conn) QueryP(dest interface{}, query string, args ...interface{}) error {
	return QueryScan(context.Background(), c, dest, query, args...)
}

func (c *Conn) QueryContextP(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return QueryScan(ctx, c, dest, query, args...)
}

type Tx struct {
	*sql.Tx
	mapper *mapper
}

func (t *Tx) Mapper() *mapper {
	return t.mapper
}

func (t *Tx) QueryP(dest interface{}, query string, args ...interface{}) error {
	return QueryScan(context.Background(), t, dest, query, args...)
}

func (t *Tx) QueryContextP(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return QueryScan(ctx, t, dest, query, args...)
}

func (t *Tx) PrepareContextP(ctx context.Context, query string) (*Stmt, error) {
	sqlStmt, err := t.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	stmt := &Stmt{Stmt: sqlStmt, mapper: t.mapper}
	return stmt, nil
}

func (t *Tx) PrepareP(query string) (*Stmt, error) {
	return t.PrepareContextP(context.Background(), query)
}

type Stmt struct {
	*sql.Stmt
	mapper *mapper
}

func (st *Stmt) Mapper() *mapper {
	return st.mapper
}

func (st *Stmt) QueryP(dest interface{}, args ...interface{}) error {
	return StmtQueryScan(context.Background(), st, dest, args...)
}

func (st *Stmt) QueryContextP(ctx context.Context, dest interface{}, args ...interface{}) error {
	return StmtQueryScan(ctx, st, dest, args...)
}

func QueryScan(ctx context.Context, qi Query, dest interface{}, query string, args ...interface{}) error {
	rows, err := qi.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return Scan(qi.Mapper(), dest, rows)
}

func StmtQueryScan(ctx context.Context, qi StmtQuery, dest interface{}, args ...interface{}) error {
	rows, err := qi.QueryContext(ctx, args...)
	if err != nil {
		return err
	}
	return Scan(qi.Mapper(), dest, rows)
}

func Scan(mapper *mapper, dest interface{}, rows *sql.Rows) error {
	defer func() {
		err := rows.Close()
		if err != nil {
			plog.WithError(err).Errorf("[porm:Scan]: rows close fail")
		}
	}()

	if mapper == nil {
		return fmt.Errorf("[porm:Scan]:mapper can not be nil")
	}
	if rows == nil {
		return nil
	}
	dv, ok := dest.(reflect.Value)
	if !ok {
		dv = reflect.ValueOf(dest)
	}
	dv = reflect.Indirect(dv)

	if dv.Kind() == reflect.Struct {
		return scanOne(mapper, dv, rows)
	}
	if dv.Kind() == reflect.Array || dv.Kind() == reflect.Slice {
		return scanSlice(mapper, dv, rows)
	}

	return fmt.Errorf("[porm:Scan]:The type of dest is not supported")
}

func scanOne(mapper *mapper, dv reflect.Value, rows *sql.Rows) error {
	if dv.Kind() != reflect.Struct {
		return fmt.Errorf("[porm:scanOne] dv must be struct, kind = %s", dv.Kind().String())
	}
	structMapper, err := mapper.Load(dv.Type())
	if err != nil {
		return err
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values, err := findValues(columns, structMapper, dv)
	if err != nil {
		return err
	}

	for rows.Next() {
		err = rows.Scan(values...)
		return err
	}
	return nil
}

func scanSlice(mapper *mapper, dv reflect.Value, rows *sql.Rows) error {
	if dv.Kind() != reflect.Slice && dv.Kind() != reflect.Array {
		return fmt.Errorf("[porm:scanSlice] dv must be array or slice, kind = %s", dv.Kind().String())
	}
	det := dv.Type().Elem()
	var ptr bool
	if det.Kind() == reflect.Ptr {
		ptr = true
		det = det.Elem()
	}

	if det.Kind() != reflect.Struct {
		return fmt.Errorf("[porm:scanSlice] det elem type must be struct, kind = %s", det.Kind().String())
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	structMapper, err := mapper.Load(det)
	if err != nil {
		return err
	}
	for rows.Next() {
		dp := reflect.New(det)
		dpe := dp.Elem()
		values, err := findValues(columns, structMapper, dpe)
		if err != nil {
			return err
		}
		err = rows.Scan(values...)
		if err != nil {
			return err
		}
		if ptr {
			dv.Set(reflect.Append(dv, dp))
		} else {
			dv.Set(reflect.Append(dv, dpe))
		}
	}

	return nil
}

func findValues(columns []string, structMapper StructMapper, dv reflect.Value) ([]interface{}, error) {
	values := make([]interface{}, len(columns))
	for index, column := range columns {
		fieldInfo, ok := structMapper.ColumnMap[column]
		if !ok {
			return nil, fmt.Errorf("[porm:findValues] column not found, column = %s", column)
		}
		values[index] = dv.FieldByIndex(fieldInfo.Index).Addr().Interface()
	}
	return values, nil
}
