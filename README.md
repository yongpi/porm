# porm
简单易用的 `go orm` 框架，练手之作。参考了 `sqlx` 和 `gorm`，整体只依赖 `github.com/yongpi/putil`
而 `putil` 只依赖 ￿￿`go sdk`， so clean !!!
## 用法介绍
### 专属 `tag`
参考了 `gorm`, 有一些结构体字段需要执行特殊逻辑，加了 `tag` 来标识。主要有
```go
const (
	Column KeyTag = iota + 1
	Readonly
	PK
)
```
### 自定义字段类型
`db` 部分参考了 `sqlx`，所以没有抽象出 `Field` 来做反射转换，依靠 `go/sql` 自带的转换方法做赋值操作。
一些传统的字段不能很好的转换，所以自定义了一些字段。**都在 `field.go` 文件里**

### 数据库相关
不想使用 `orm` 可以用 `DB` + `psql` 也能方便的进行CURD操作 （`DB` 参考了 `sqlx`）。
`orm` 支持了单库和单主多从的模式，但是也可以自己扩展，只需要实现 `Storage` 接口即可

### 事务传播
可以手动开启事务：`orm.BeginTx`，也可以使用 `orm.Transaction` 传入函数来执行事务操作。
在 `orm.Transaction` 内，事务会以 `context` 为载体进行传播，只支持单事务模式。

### 例子
```go
package porm

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yongpi/putil/psql"
)

func init() {
	simpleConfig := SimpleStorageConfig{
		DriverName:     "mysql",
		DataSourceName: "root:peiyongkang@tcp(127.0.0.1:3306)/test?charset=utf8",
		HolderType:     psql.Question,
		StorageName:    "test",
	}

	RegisterSimpleStorage(simpleConfig)

	var list []*SimpleStorageConfig
	mc := SimpleStorageConfig{
		DriverName:     "mysql",
		DataSourceName: "root:peiyongkang@tcp(127.0.0.1:3306)/test?charset=utf8",
		HolderType:     psql.Question,
	}
	list = append(list, &mc)

	sc := SimpleStorageConfig{
		DriverName:     "mysql",
		DataSourceName: "root:peiyongkang@tcp(127.0.0.1:3306)/test?charset=utf8",
		HolderType:     psql.Question,
	}
	list = append(list, &sc)

	sc2 := SimpleStorageConfig{
		DriverName:     "mysql",
		DataSourceName: "root:peiyongkang@tcp(127.0.0.1:3306)/test?charset=utf8",
		HolderType:     psql.Question,
	}
	list = append(list, &sc2)

	msc := MasterSlaveStorageConfig{StorageName: "ms", DBConfigs: list}

	RegisterMasterSlaveStorage(msc)
}

type AlbumAuthorModel struct {
	ID        int64 `porm:"pk"`
	Name      string
	Artwork   string
	Bio       string
	MemberID  NullInt64
	Career    string
	CreatedAt Time
	UpdatedAt Time `porm:"readonly"`
}

func (m *AlbumAuthorModel) TableName() string {
	return "album_author"
}

func TestSimple(t *testing.T) {
	var m AlbumAuthorModel
	ctx := context.Background()
	st := psql.Select("*").Where(psql.Eq{"id": 1})

	err := ORM().Select(ctx, st, &m)
	if err != nil {
		t.Error(err)
	}

	var list []*AlbumAuthorModel
	st = psql.Select("*").Where(psql.Eq{"id": []int64{1, 2}})
	err = ORM().Select(ctx, st, &list)
	if err != nil {
		t.Error(err)
	}

	m.Name = "up"
	m.MemberID.SetInt64(123)
	m.CreatedAt.SetTime(time.Now())

	result, err := ORM().UpdateModel(ctx, &m)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(result.RowsAffected())

	//m.ID = 3
	//m.CreatedAt.BeNull()
	//
	//result, err = ORM().Insert(ctx, &m)
	//if err != nil {
	//	t.Error(err)
	//}
	//fmt.Println(result.RowsAffected())

	list[0].ID = 4
	list[1].ID = 5

	result, err = ORM().Insert(ctx, &list)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(result.RowsAffected())

}

func TestMasterSlave(t *testing.T) {
	var m AlbumAuthorModel
	ctx := context.Background()
	st := psql.Select("*").Where(psql.Eq{"id": 1})

	err := NORM("ms").Select(ctx, st, &m)
	if err != nil {
		t.Error(err)
	}

	m.Name = "up111"
	m.MemberID.SetInt64(123)
	m.CreatedAt.SetTime(time.Now())

	result, err := NORM("ms").UpdateModel(ctx, &m)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(result.RowsAffected())

}

func TestOrm_Transaction(t *testing.T) {
	ctx := context.Background()
	err := ORM().Transaction(ctx, func(ctx context.Context, orm *orm) error {
		var m AlbumAuthorModel

		st := psql.Select("*").Where(psql.Eq{"id": 1})
		err := orm.Select(ctx, st, &m)
		if err != nil {
			return err
		}

		m.MemberID.SetInt64(1111)
		_, err = orm.UpdateModel(ctx, &m)
		if err != nil {
			return err
		}

		_, err = ORM().Insert(ctx, &m)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		t.Error(err)
	}
}

```