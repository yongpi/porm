package porm

import (
	"database/sql"
	"time"
)

const (
	SqlTimeFormat = "2006-01-02 15:04:05"
)

func CoverNullValue(value interface{}) interface{} {
	if nv, ok := value.(NullValue); ok {
		return nv.NullInterface()
	}
	return value
}

type NullValue interface {
	NullInterface() interface{}
	BeNull()
}

type NullInt64 struct {
	sql.NullInt64
}

func (n *NullInt64) SetInt64(value int64) {
	n.Int64 = value
	n.Valid = true
}

func (n NullInt64) NullInterface() interface{} {
	if n.Valid {
		return n.Int64
	}
	return nil
}

func (n NullInt64) BeNull() {
	n.Valid = false
}

type NullInt32 struct {
	sql.NullInt32
}

func (n NullInt32) NullInterface() interface{} {
	if n.Valid {
		return n.Int32
	}
	return nil
}

func (n NullInt32) BeNull() {
	n.Valid = false
}

func (n *NullInt32) SetInt32(value int32) {
	n.Int32 = value
	n.Valid = true
}

type NullString struct {
	sql.NullString
}

func (n NullString) NullInterface() interface{} {
	if n.Valid {
		return n.String
	}
	return nil
}

func (n NullString) BeNull() {
	n.Valid = false
}

func (n *NullString) SetString(value string) {
	n.String = value
	n.Valid = true
}

type NullBool struct {
	sql.NullBool
}

func (n NullBool) NullInterface() interface{} {
	if n.Valid {
		return n.Bool
	}
	return nil
}

func (n NullBool) BeNull() {
	n.Valid = false
}

func (n *NullBool) SetBool(value bool) {
	n.Bool = true
	n.Valid = true
}

type NullFloat64 struct {
	sql.NullFloat64
}

func (n NullFloat64) NullInterface() interface{} {
	if n.Valid {
		return n.Float64
	}
	return nil
}

func (n NullFloat64) BeNull() {
	n.Valid = false
}

func (n *NullFloat64) SetFloat64(value float64) {
	n.Float64 = value
	n.Valid = true
}

type Time struct {
	time.Time
	Valid bool
}

func (t *Time) SetTime(time time.Time) {
	t.Time = time
}

func (t Time) NullInterface() interface{} {
	if !t.Valid || t.IsZero() {
		return nil
	}

	return t.Time
}

func (t Time) BeNull() {
	t.Valid = false
}

func (t *Time) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch st := src.(type) {
	case []byte:
		pt, err := time.Parse(SqlTimeFormat, string(st))
		if err != nil {
			return err
		}
		t.Time = pt
	case time.Time:
		t.Time = st
	}

	return nil
}
