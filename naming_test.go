package porm

import (
	"testing"
)

func TestHumpNamingFunc(t *testing.T) {
	a := "ID"
	if HumpNamingFunc(a) != "id" {
		t.Errorf("hump naming func not expected, value = %s", a)
	}

	a = "FuncName"
	if HumpNamingFunc(a) != "func_name" {
		t.Errorf("hump naming func not expected, value = %s", a)
	}

	a = "FFFFuuuuAAAAAAABBBaaaCccc"
	if HumpNamingFunc(a) != "ffffuuuu_aaaaaaabbbaaa_cccc" {
		t.Errorf("hump naming func not expected, value = %s", a)
	}

	a = "titleName"
	if HumpNamingFunc(a) != "title_name" {
		t.Errorf("hump naming func not expected, value = %s", a)
	}
}
