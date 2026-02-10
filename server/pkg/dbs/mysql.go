package dbs

import (
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"xorm.io/xorm"
)

var (
	DBAdmin *XormDB
)

type MysqlCfg struct {
	Addr        string
	UserName    string
	Password    string
	DB          string
	CharSet     string
	Maxopen     int
	Maxidle     int
	MaxIdleTime string
	MaxLifeTime string
}

func InitMysql(conCfg *MysqlCfg, db **XormDB) {
	logx.Infof("mysql config: %+v", conCfg)

	engine, err := xorm.NewEngine("mysql", conCfg.ConnAddr())
	if err != nil {
		panic(err)
	}
	engine.SetMaxOpenConns(conCfg.Maxopen)
	engine.SetMaxIdleConns(conCfg.Maxidle)
	maxIdleTime, _ := time.ParseDuration(conCfg.MaxIdleTime)
	engine.SetConnMaxIdleTime(maxIdleTime)
	maxLifeTime, _ := time.ParseDuration(conCfg.MaxLifeTime)
	engine.SetConnMaxLifetime(maxLifeTime)

	timer := time.AfterFunc(time.Second*3, func() {
		panic(fmt.Sprintf("mysql conn timeout, addr: %s", conCfg.ConnAddr()))
	})
	err = engine.Ping()
	if err != nil {
		panic(err)
	}
	timer.Stop()
	timer = nil
	*db = &XormDB{engine}
}

func IsTableNotExistError(err error) bool {
	if sqlErr, ok := err.(*mysql.MySQLError); ok {
		return sqlErr.Number == 1146
	}
	return false
}

func (cfg *MysqlCfg) ConnAddr() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&loc=Local", cfg.UserName, cfg.Password, cfg.Addr, cfg.DB, cfg.CharSet)
}

type XormDB struct {
	*xorm.Engine
}

// 事务封装操作
func (x XormDB) WithTx(txFunc func(session *xorm.Session) (err error)) error {
	session := x.Engine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return err
	}

	err := txFunc(session)
	if err != nil {
		err2 := session.Rollback()
		if err2 != nil {
			logx.Errorf("rollback error %v", err2)
		}
		return err
	}

	return session.Commit()
}
