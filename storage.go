package porm

import (
	"sync"
	"sync/atomic"

	"github.com/yongpi/putil/psql"
)

type Storage interface {
	GetDB(orm *orm) *DB
	GetName() string
	SqlBuilder() psql.SqlBuilder
	GetMapper() *mapper
}

var (
	defaultStorage Storage
	ormOnce        sync.Once
	depository     = make(map[string]Storage)
)

func RegisterStorage(storage Storage) {
	ormOnce.Do(func() {
		defaultStorage = storage
	})

	_, ok := depository[storage.GetName()]
	if !ok {
		depository[storage.GetName()] = storage
	}
}

func RegisterSimpleStorage(config SimpleStorageConfig) {
	db, err := OpenDBName(config.DriverName, config.DataSourceName, config.StorageName)
	if err != nil {
		panic(err)
	}
	sqlBuilder := psql.NewSqlBuilder(config.HolderType)
	storage := &SimpleStorage{db: db, sqlBuilder: &sqlBuilder, Name: config.StorageName}
	RegisterStorage(storage)
}

type SimpleStorageConfig struct {
	DriverName     string
	DataSourceName string
	StorageName    string
	HolderType     psql.PlaceHolderType
}

type SimpleStorage struct {
	db         *DB
	sqlBuilder *psql.SqlBuilder
	Name       string
}

func (s *SimpleStorage) GetDB(orm *orm) *DB {
	return s.db
}

func (s *SimpleStorage) GetName() string {
	return s.db.Name
}

func (s *SimpleStorage) SqlBuilder() psql.SqlBuilder {
	return *s.sqlBuilder
}

func (s *SimpleStorage) GetMapper() *mapper {
	return s.db.Mapper()
}

type MasterSlaveStorage struct {
	master     *SimpleStorage
	slaves     []*SimpleStorage
	count      int64
	sqlBuilder psql.SqlBuilder
}

func (s *MasterSlaveStorage) GetDB(orm *orm) *DB {
	return s.pick(orm).GetDB(orm)
}

func (s *MasterSlaveStorage) GetName() string {
	return s.master.Name
}

func (s *MasterSlaveStorage) SqlBuilder() psql.SqlBuilder {
	return s.sqlBuilder
}

func (s *MasterSlaveStorage) GetMapper() *mapper {
	return s.master.GetMapper()
}

func (s *MasterSlaveStorage) RoundRobinSlave() Storage {
	index := s.count % int64(len(s.slaves))
	slave := s.slaves[index]
	atomic.AddInt64(&s.count, 1)
	return slave
}

func (s *MasterSlaveStorage) pick(orm *orm) Storage {
	if orm.forceMaster || orm.sqlAction != Select || len(s.slaves) == 0 {
		return s.master
	}
	return s.RoundRobinSlave()
}

type MasterSlaveStorageConfig struct {
	StorageName string
	HolderType  psql.PlaceHolderType
	DBConfigs   []*SimpleStorageConfig
}

func RegisterMasterSlaveStorage(config MasterSlaveStorageConfig) {
	ms := &MasterSlaveStorage{}
	storageName := config.StorageName
	for index, cf := range config.DBConfigs {
		db, err := OpenDBName(cf.DriverName, cf.DataSourceName, storageName)
		if err != nil {
			panic(err)
		}
		ss := &SimpleStorage{db: db, Name: storageName}
		if index == 0 {
			ms.master = ss
		}

		ms.slaves = append(ms.slaves, ss)
	}
	ms.sqlBuilder = psql.NewSqlBuilder(config.HolderType)

	RegisterStorage(ms)
}
