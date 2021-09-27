package porm

import (
	"fmt"
	"reflect"

	"github.com/yongpi/putil/psql"
)

type SqlNeed struct {
	Columns   []string
	TableName string
}

func FillSelect(mapper *mapper, st *psql.SelectTransform, model interface{}, holderType psql.PlaceHolderType) error {
	if len(st.Columns) == 0 || st.Columns[0] == "*" {
		columns, err := PickUpColumns(mapper, model)
		if err != nil {
			return err
		}
		st.Columns = columns
	}

	if st.TableName == "" {
		tableName, err := PickUpTable(model)
		if err != nil {
			return err
		}

		st.TableName = tableName
	}

	st.HolderType = holderType

	return nil
}

func FillInsert(mapper *mapper, st *psql.InsertTransform, model interface{}, holderType psql.PlaceHolderType) error {
	st.HolderType = holderType

	value := reflect.Indirect(reflect.ValueOf(model))
	if value.Kind() == reflect.Struct {
		table, ok := model.(Model)
		if !ok {
			return fmt.Errorf("[porm:FillInsert]: model must implement Model interface")
		}
		st.Table(table.TableName())
		return BuilderInsertOne(mapper, st, value)
	}
	return BuilderInsertList(mapper, st, value)
}

func BuilderInsertOne(mapper *mapper, st *psql.InsertTransform, value reflect.Value) error {
	sm, err := mapper.Load(value.Type())
	if err != nil {
		return err
	}
	var values []interface{}
	for _, column := range sm.Columns {
		if column.ReadOnly {
			continue
		}
		cv := value.FieldByIndex(column.Index)
		if column.PK {
			if !cv.IsValid() || cv.IsZero() {
				continue
			}
		}

		st.Column(column.Name)
		values = append(values, CoverNullValue(cv.Interface()))
	}
	st.Value(values...)
	return nil
}

func BuilderInsertList(mapper *mapper, st *psql.InsertTransform, value reflect.Value) error {
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return fmt.Errorf("[porm:BuilderInsertList] model must be array or slice")
	}

	met := value.Type().Elem()
	if met.Kind() == reflect.Ptr {
		met = met.Elem()
	}

	table, ok := reflect.New(met).Interface().(Model)
	if !ok {
		return fmt.Errorf("[porm:BuilderInsertList] model must implement Model interface")
	}

	st.Table(table.TableName())
	sm, err := mapper.Load(met)
	if err != nil {
		return err
	}

	var (
		fi      []interface{}
		columns []string
	)
	first := reflect.Indirect(value.Index(0))

	for _, column := range sm.Columns {
		if column.ReadOnly {
			continue
		}
		cv := first.FieldByIndex(column.Index)
		if column.PK {
			if !cv.IsValid() || cv.IsZero() {
				continue
			}
		}

		columns = append(columns, column.Name)
		fi = append(fi, CoverNullValue(cv.Interface()))
	}
	st.Value(fi...)

	st.Column(columns...)

	for i := 1; i < value.Len(); i++ {
		ve := reflect.Indirect(value.Index(i))
		vi := make([]interface{}, len(columns))

		for index, key := range columns {
			column := sm.ColumnMap[key]
			vi[index] = CoverNullValue(ve.FieldByIndex(column.Index).Interface())
		}
		st.Value(vi...)
	}

	return nil
}

func FillUpdate(st *psql.UpdateTransform, model interface{}, holderType psql.PlaceHolderType) error {
	if st.TableName == "" {
		tableName, err := PickUpTable(model)
		if err != nil {
			return err
		}

		st.TableName = tableName
	}

	st.HolderType = holderType

	return nil
}

func FillDelete(st *psql.DeleteTransform, model interface{}, holderType psql.PlaceHolderType) error {
	if st.TableName == "" {
		tableName, err := PickUpTable(model)
		if err != nil {
			return err
		}

		st.TableName = tableName
	}

	st.HolderType = holderType
	return nil
}

func PickUpColumns(mapper *mapper, model interface{}) ([]string, error) {
	mv := reflect.Indirect(reflect.ValueOf(model))
	if mv.Kind() == reflect.Struct {
		return mapper.Columns(mv.Type())
	}

	if mv.Kind() != reflect.Slice && mv.Kind() != reflect.Array {
		return nil, fmt.Errorf("[porm:fillList] model must be array or slice")
	}

	met := mv.Type().Elem()
	if met.Kind() == reflect.Ptr {
		met = met.Elem()
	}

	return mapper.Columns(met)
}

func PickUpTable(model interface{}) (string, error) {
	mv := reflect.Indirect(reflect.ValueOf(model))
	if mv.Kind() == reflect.Struct {
		table, ok := model.(Model)
		if !ok {
			return "", fmt.Errorf("[porm:PickUpTable] model must implement Model interface")
		}
		return table.TableName(), nil
	}

	if mv.Kind() != reflect.Slice && mv.Kind() != reflect.Array {
		return "", fmt.Errorf("[porm:PickUpTable] model must be array or slice")
	}

	met := mv.Type().Elem()
	if met.Kind() == reflect.Ptr {
		met = met.Elem()
	}

	table, ok := reflect.New(met).Interface().(Model)
	if !ok {
		return "", fmt.Errorf("[porm:PickUpTable] model must implement Model interface")
	}
	return table.TableName(), nil
}
