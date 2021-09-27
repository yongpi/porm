package porm

import (
	"encoding/json"
	"testing"
)

type TestM1 struct {
	Name string
	Age  int
}

type TestM2 struct {
	Title string
	Sex   bool
	TestM1
}

type TestM struct {
	Description string
	ID          int `porm:"pk"`
	Zone        string
	TestM2
	BasicLa   string `porm:"column:basicla,readonly"`
	UpdatedAt int64  `porm:"readonly"`
}

func TestMapping(t *testing.T) {
	mapper := NewMapper("test")
	data, err := mapper.Load(TestM{})
	if err != nil {
		t.Error(err)
	}

	js, err := json.Marshal(data.Columns)
	if err != nil {
		t.Error(err)
	}

	exJs := `[{"Name":"description","Index":[0],"PK":false,"ReadOnly":false},{"Name":"id","Index":[1],"PK":true,"ReadOnly":false},{"Name":"zone","Index":[2],"PK":false,"ReadOnly":false},{"Name":"title","Index":[3,0],"PK":false,"ReadOnly":false},{"Name":"sex","Index":[3,1],"PK":false,"ReadOnly":false},{"Name":"name","Index":[3,2,0],"PK":false,"ReadOnly":false},{"Name":"age","Index":[3,2,1],"PK":false,"ReadOnly":false},{"Name":"basicla","Index":[4],"PK":false,"ReadOnly":true},{"Name":"updated_at","Index":[5],"PK":false,"ReadOnly":true}]`
	if string(js) != exJs {
		t.Errorf("mapping fail, mapping = %s", string(js))
	}

	for _, column := range data.Columns {
		_, ok := data.ColumnMap[column.Name]
		if !ok {
			t.Errorf("column not exist map, column = %s", column.Name)
		}
	}
}
