package porm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/yongpi/putil/plog"

	"github.com/yongpi/putil/psql"
)

type SqlAction int

const (
	Select SqlAction = iota
	Insert
	Update
	Delete
)

type Model interface {
	TableName() string
}

func ORM() *orm {
	orm := &orm{storage: defaultStorage}
	return orm
}

func NORM(storageName string) *orm {
	orm := &orm{storage: depository[storageName]}
	return orm
}

type orm struct {
	storage     Storage
	sqlAction   SqlAction
	forceMaster bool
	tx          *Tx
	err         error
	inTx        bool
}

func (o *orm) Copy(dest *orm) {
	o.storage = dest.storage
	o.tx = dest.tx
	o.forceMaster = dest.forceMaster
	o.inTx = dest.inTx
	o.err = dest.err
}

func (o *orm) BeginTx(ctx context.Context) (*orm, error) {
	to := TxORMFromContext(ctx)
	if to != nil && to.StorageName() == o.StorageName() {
		plog.Infof("[porm:orm:BeginTx]: begin tx from context, db = %s", to.StorageName())
		return to, nil
	}

	o.ForceMaster()
	tx, err := o.DB().BeginTxP(ctx, nil)
	if err != nil {
		return nil, err
	}
	o.tx = tx

	plog.Infof("[porm:orm:BeginTx]: begin tx, db = %s", o.StorageName())
	return o, nil
}

func (o *orm) Commit() error {
	if o.tx == nil {
		return fmt.Errorf("[porm:orm:Commit]:orm tx can not be nil")
	}

	plog.Infof("[porm:orm:Commit]: commit tx, db = %s", o.StorageName())
	return o.tx.Commit()
}

func (o *orm) RollBack() error {
	if o.tx == nil {
		return fmt.Errorf("[porm:orm:RollBack]:orm tx can not be nil")
	}

	plog.Infof("[porm:orm:RollBack]: rollback tx, db = %s", o.StorageName())
	return o.tx.Rollback()
}

func (o *orm) MustRollBack() {
	if o.tx == nil {
		panic("[porm:orm:MustRollBack]:orm tx can not be nil")
	}

	plog.Infof("[porm:orm:MustRollBack]: rollback tx, db = %s", o.StorageName())
	err := o.tx.Rollback()
	if err != nil {
		panic(fmt.Errorf("[porm:orm:MustRollBack]: rollback error, err = %s", err.Error()))
	}
}

func (o *orm) Transaction(ctx context.Context, fun func(ctx context.Context, orm *orm) error) error {
	no, err := o.BeginTx(ctx)
	if err != nil {
		return err
	}

	no.inTx = true

	ctx = WithTxContext(ctx, no)
	defer WithTxContext(ctx, nil)
	defer func() {
		if err := recover(); err != nil {
			no.MustRollBack()
			panic(err)
		}
	}()

	err = fun(ctx, no)
	if err != nil {
		err2 := no.RollBack()
		if err2 != nil {
			return fmt.Errorf("[porm:orm:Transaction]: rollback fail, err = %s", err2)
		}
		return err
	}

	return no.Commit()
}

func (o *orm) DB() *DB {
	return o.storage.GetDB(o)
}

func (o *orm) Mapper() *mapper {
	return o.storage.GetMapper()
}

func (o *orm) StorageName() string {
	return o.storage.GetName()
}

func (o *orm) SqlBuilder() psql.SqlBuilder {
	return o.storage.SqlBuilder()
}

func (o *orm) ForceMaster() *orm {
	o.forceMaster = true
	return o
}

func (o *orm) Select(ctx context.Context, st *psql.SelectTransform, model interface{}) error {
	o.sqlAction = Select
	if nil == st {
		return fmt.Errorf("[porm:orm:Select] st can not be nil")
	}

	err := FillSelect(o.Mapper(), st, model, o.SqlBuilder().HolderType)
	if err != nil {
		return err
	}

	query, args, err := st.ToSql()
	if err != nil {
		return err
	}
	// 打印日志
	plog.Debugf("[porm:orm:Select]: query sql = %s, args = %#v", query, args)

	// 执行 hook
	Fishing(ctx, BeforeSelect, o)
	if o.err != nil {
		return o.err
	}

	defer Fishing(ctx, AfterSelect, o)

	var stmt *Stmt
	if o.tx != nil {
		stmt, err = o.tx.PrepareContextP(ctx, query)
	} else {
		stmt, err = o.DB().PrepareContextP(ctx, query)
	}
	if err != nil {
		o.err = err
		return o.err
	}
	defer func() {
		err := stmt.Close()
		plog.WithError(err).Error("[porm:orm:Select]: stmt close fail")
	}()

	err = stmt.QueryContextP(ctx, model, args...)
	if err != nil {
		o.err = err
		return o.err
	}

	return o.err
}

func (o *orm) SelectX(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	o.sqlAction = Select

	// 执行 hook
	Fishing(ctx, BeforeSelect, o)
	if o.err != nil {
		return nil, o.err
	}

	defer Fishing(ctx, AfterSelect, o)

	// 打印日志
	plog.Debugf("[porm:orm:SelectX]: query sql = %s, args = %#v", query, args)

	var stmt *Stmt
	var err error
	if o.tx != nil {
		stmt, err = o.tx.PrepareContextP(ctx, query)
	} else {
		stmt, err = o.DB().PrepareContextP(ctx, query)
	}
	if err != nil {
		o.err = err
		return nil, o.err
	}
	defer func() {
		err := stmt.Close()
		if err != nil {
			plog.WithError(err).Error("[porm:orm:SelectX]: stmt close fail")
		}
	}()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		o.err = err
		return nil, o.err
	}

	return rows, o.err
}

func (o *orm) exec(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	var stmt *Stmt
	if o.tx != nil {
		stmt, err = o.tx.PrepareContextP(ctx, query)
	} else {
		stmt, err = o.DB().PrepareContextP(ctx, query)
	}
	if err != nil {
		o.err = err
		return nil, o.err
	}

	defer func() { _ = stmt.Close() }()

	result, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		o.err = err
		return nil, o.err
	}

	return result, nil

}

func (o *orm) Update(ctx context.Context, st *psql.UpdateTransform, model interface{}) (sql.Result, error) {
	if st == nil {
		return nil, fmt.Errorf("[porm:orm:Update] st can not be nil")
	}

	err := FillUpdate(st, model, o.SqlBuilder().HolderType)
	if err != nil {
		return nil, err
	}

	query, args, err := st.ToSql()
	if err != nil {
		return nil, err
	}

	return o.UpdateX(ctx, query, args...)
}

func (o *orm) UpdateX(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	o.sqlAction = Update

	// 打印日志
	plog.Debugf("[porm:orm:UpdateX]: query sql = %s, args = %#v", query, args)

	// 执行 hook
	Fishing(ctx, BeforeUpdate, o)
	if o.err != nil {
		return nil, o.err
	}

	defer Fishing(ctx, AfterUpdate, o)

	return o.exec(ctx, query, args...)
}

func (o *orm) UpdateModel(ctx context.Context, model interface{}) (sql.Result, error) {
	table, ok := model.(Model)
	if !ok {
		return nil, fmt.Errorf("[porm:orm:UpdateModel]: model must implement Model interface")
	}

	value := reflect.Indirect(reflect.ValueOf(model))
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("[porm:orm:UpdateModel]: value must be struct")
	}
	sm, err := o.Mapper().Load(value.Type())
	if err != nil {
		return nil, err
	}
	st := o.SqlBuilder().Update(table.TableName())
	for _, column := range sm.Columns {
		if column.PK {
			st.Where(psql.Eq{column.Name: value.FieldByIndex(column.Index).Interface()})
			continue
		}
		if column.ReadOnly {
			continue
		}
		st.Set(column.Name, CoverNullValue(value.FieldByIndex(column.Index).Interface()))
	}

	query, args, err := st.ToSql()
	if err != nil {
		return nil, err
	}

	return o.UpdateX(ctx, query, args...)
}

func (o *orm) Delete(ctx context.Context, st *psql.DeleteTransform, model interface{}) (sql.Result, error) {
	if st == nil {
		return nil, fmt.Errorf("[porm:orm:Delete] st can not be nil")
	}

	err := FillDelete(st, model, o.SqlBuilder().HolderType)
	if err != nil {
		return nil, err
	}

	query, args, err := st.ToSql()
	if err != nil {
		return nil, err
	}
	return o.DeleteX(ctx, query, args...)
}

func (o *orm) DeleteX(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	o.sqlAction = Delete

	// 打印日志
	plog.Debugf("[porm:orm:DeleteX]: query sql = %s, args = %#v", query, args)

	// 执行 hook
	Fishing(ctx, BeforeDelete, o)
	if o.err != nil {
		return nil, o.err
	}

	defer Fishing(ctx, AfterDelete, o)

	return o.exec(ctx, query, args...)
}

func (o *orm) InsertX(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	o.sqlAction = Insert

	// 打印日志
	plog.Debugf("[porm:orm:InsertX]: query sql = %s, args = %#v", query, args)

	// 执行 hook
	Fishing(ctx, BeforeInsert, o)
	if o.err != nil {
		return nil, o.err
	}

	defer Fishing(ctx, AfterInsert, o)

	return o.exec(ctx, query, args...)
}

func (o *orm) Insert(ctx context.Context, model interface{}) (sql.Result, error) {
	st := psql.NewInsert(o.SqlBuilder().HolderType)
	err := FillInsert(o.Mapper(), st, model, o.SqlBuilder().HolderType)
	if err != nil {
		return nil, err
	}

	query, args, err := st.ToSql()
	if err != nil {
		return nil, err
	}

	return o.InsertX(ctx, query, args...)

}
