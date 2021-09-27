package porm

import (
	"reflect"
	"strings"
)

const TagPrefix = "porm"

var emptyTagInfo TagInfo

type KeyTag int

const (
	Column KeyTag = iota + 1
	Readonly
	PK
)

func (t KeyTag) String() string {
	switch t {
	case Column:
		return "column"
	case Readonly:
		return "readonly"
	case PK:
		return "pk"
	}

	return ""
}

type TagInfo struct {
	HasColumn bool
	Column    string
	Readonly  bool
	PK        bool
}

func LookUp(st reflect.StructTag) TagInfo {
	data, ok := st.Lookup(TagPrefix)
	if !ok {
		return emptyTagInfo
	}

	list := strings.Split(data, ",")
	var info TagInfo
	for _, item := range list {
		kv := strings.Split(item, ":")
		key := kv[0]
		switch key {
		case Column.String():
			if len(kv) == 2 {
				info.HasColumn = true
				info.Column = kv[1]
			}
		case Readonly.String():
			info.Readonly = true
		case PK.String():
			info.PK = true
		}
	}

	return info
}
