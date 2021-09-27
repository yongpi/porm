package porm

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	mappers     sync.Map
	emptyStruct StructMapper
)

func NewMapper(DBName string) *mapper {
	return NewMapperFunc(DBName, DefaultNamingFunc)
}

func NewMapperFunc(DBName string, NameFunc NamingFunc) *mapper {
	data, ok := mappers.Load(DBName)
	if ok {
		return data.(*mapper)
	}

	mp := &mapper{DBName: DBName, NameFunc: NameFunc}
	mappers.Store(DBName, mp)
	return mp
}

type mapper struct {
	Cache    sync.Map
	DBName   string
	NameFunc NamingFunc
}

type FieldInfo struct {
	Name     string
	Index    []int
	PK       bool
	ReadOnly bool
}

type StructMapper struct {
	Columns   []*FieldInfo
	ColumnMap map[string]*FieldInfo
}

func (m *StructMapper) AddColumn(column *FieldInfo) {
	m.Columns = append(m.Columns, column)
	m.ColumnMap[column.Name] = column
}

func (m *mapper) Load(value interface{}) (StructMapper, error) {
	vt, ok := value.(reflect.Type)
	if !ok {
		vt = reflect.TypeOf(value)
	}
	if vt.Kind() != reflect.Struct {
		return emptyStruct, fmt.Errorf("load mapper value type must be struct, kind = %s", vt.Kind().String())
	}

	data, ok := m.Cache.Load(vt)
	if ok {
		return data.(StructMapper), nil
	}

	sd, err := m.store(vt)
	if err != nil {
		return emptyStruct, err
	}

	return sd, nil
}
func (m *mapper) Columns(value reflect.Type) ([]string, error) {
	vt, err := m.Load(value)
	if err != nil {
		return nil, err
	}

	var columns []string
	for _, column := range vt.Columns {
		columns = append(columns, column.Name)
	}

	return columns, nil
}

func (m *mapper) store(value reflect.Type) (StructMapper, error) {
	if value.Kind() != reflect.Struct {
		return emptyStruct, fmt.Errorf("store mapping value must be struct, kind = %s", value.Kind().String())
	}

	mapper := &StructMapper{ColumnMap: make(map[string]*FieldInfo)}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		mapper = m.parseField(field, mapper, nil)
	}

	m.Cache.Store(value, *mapper)
	return *mapper, nil
}

func (m *mapper) parseField(field reflect.StructField, mapper *StructMapper, index []int) *StructMapper {
	// 非导出字段不处理
	if field.PkgPath != "" && !field.Anonymous {
		return mapper
	}

	if field.Anonymous && field.Type.Kind() == reflect.Struct {
		index = append(index, field.Index...)
		for i := 0; i < field.Type.NumField(); i++ {
			mapper = m.parseField(field.Type.Field(i), mapper, index)
		}

		return mapper
	}

	// 普通字段
	tagInfo := LookUp(field.Tag)
	var column FieldInfo
	column.Index = append(column.Index, index...)
	column.Index = append(column.Index, field.Index...)

	if !tagInfo.HasColumn {
		column.Name = m.NameFunc(field.Name)
	} else {
		column.Name = tagInfo.Column
	}
	column.ReadOnly = tagInfo.Readonly
	column.PK = tagInfo.PK

	mapper.AddColumn(&column)
	return mapper

}
