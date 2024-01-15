// Package db 数据库访问
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go/log"

	_ "github.com/go-sql-driver/mysql" // 隐藏依赖包的实现
	"github.com/jmoiron/sqlx"
)

const ProtoMysql = "mysql"

// 解析配置
type config struct {
	Custom struct {
		Client []struct {
			Namespace   string `yaml:"namespace"`
			Target      string `yaml:"target"`
			Protocol    string `yaml:"protocol"`
			MaxLifetime int    `yaml:"max_lifetime"`
			MaxIdle     int    `yaml:"max_dile"`
			MaxOpen     int    `yaml:"max_open"`
		} `yaml:"client"`
	} `yaml:"custom"`
}

// MysqlDbProxy 是mysql接口
type MysqlDbProxy interface {
	GetNamespace() []string

	GetDB(ns string) (*sqlx.DB, error)

	InsertCompareLog(neList []string, taskId string,
		compareStatus int8, compareType int8, compareItem int8) error

	UpdateCompareLogStatus(neList []string, taskId string, item int8, status int8) error

	CountCompareLog() (total int64, err error)

	GetCompareLog(c context.Context, pn int32, rn int32) ([]*CompareLog, error)

	GetCompareLogByTaskId(taskId string) ([]string, error)
}

// MysqlDbProxy 数据库对象
type mysqlDbProxy struct {
	dbs map[string]*sqlx.DB
}

// NewMysqlDbProxy NewDbProxy 初始化数据库对象
func NewMysqlDbProxy(confPath string) (MysqlDbProxy, error) {
	proxy := &mysqlDbProxy{dbs: make(map[string]*sqlx.DB)}

	content, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Errorf("get config file failed: %v ", err)
		return nil, err
	}
	conf := new(config)
	if err := yaml.Unmarshal(content, conf); err != nil {
		log.Errorf("get db config failed: %v", err)
		return nil, err
	}

	for _, v := range conf.Custom.Client {
		log.Info(v.Namespace)
		if v.Protocol != ProtoMysql {
			continue
		}
		mydb, err := sql.Open("mysql", v.Target)
		if err != nil {
			log.Errorf("open db failed: %v", err)
			return nil, err
		}
		mydb.SetMaxOpenConns(v.MaxOpen)
		mydb.SetMaxIdleConns(v.MaxIdle)
		mydb.SetConnMaxLifetime(time.Duration(v.MaxLifetime) * time.Millisecond)
		proxy.dbs[v.Namespace] = sqlx.NewDb(mydb, "mysql")
		runtime.SetFinalizer(mydb, func(mydb *sql.DB) { mydb.Close() })
	}

	/*
		sqls := strings.Split(schema, "$")
		for _, sql := range sqls {
			if _, err = proxy.mydb.Exec(sql); err != nil {
				log.Errorf("create table failed(%v)", err)
				return nil, err
			}
		}
	*/
	return proxy, nil
}

func (proxy *mysqlDbProxy) GetNamespace() []string {
	var res []string
	for k, _ := range proxy.dbs {
		res = append(res, k)
	}
	return res
}

// GetDB 获取数据库对象指针
func (proxy *mysqlDbProxy) GetDB(ns string) (*sqlx.DB, error) {
	inst, ok := proxy.dbs[ns]
	if !ok {
		return nil, fmt.Errorf("not found db instance of %s", ns)
	}
	return inst, nil
}

func (proxy *mysqlDbProxy) InsertCompareLog(neList []string, taskId string,
	compareStatus int8, compareType int8, compareItem int8) error {
	dbCon, ok := proxy.dbs["mysql_online"]
	if !ok {
		return fmt.Errorf("not found mysql client of %s", "mysql_online")
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	currentTime := time.Now()
	createTime := time.Date(
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day(),
		currentTime.Hour(),
		currentTime.Minute(),
		currentTime.Second(),
		0,
		currentTime.In(loc).Location()).Unix()

	for i := 0; i < len(neList); i++ {
		sqlStr := fmt.Sprintf(
			`INSERT INTO compare_log (ne_id, task_id, create_time, status, type, item)  
				 VALUES ("%s","%s",%d,%d,%d,%d)`,
			neList[i],
			taskId,
			createTime,
			compareStatus,
			compareType,
			compareItem)
		log.Info(sqlStr)

		if _, err := dbCon.Exec(sqlStr); err != nil {
			log.Errorf("insert neId[%s] into compare_log failed: %s", neList[i], err)
			return err
		}
	}
	log.Infof("sync compare_log[%s] success done", neList)

	return nil
}

func (proxy *mysqlDbProxy) UpdateCompareLogStatus(neList []string, taskId string,
	item int8, status int8) error {
	dbCon, ok := proxy.dbs["mysql_online"]
	if !ok {
		return fmt.Errorf("not found mysql client of %s", "mysql_online")
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	currentTime := time.Now()
	endTime := time.Date(
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day(),
		currentTime.Hour(),
		currentTime.Minute(),
		currentTime.Second(),
		0,
		currentTime.In(loc).Location()).Unix()

	for i := 0; i < len(neList); i++ {
		sqlStr := fmt.Sprintf(
			`UPDATE compare_log SET	status=%d, end_time=%d WHERE task_id="%s" AND ne_id="%s" AND item=%d`,
			status,
			endTime,
			taskId,
			neList[i],
			item)
		log.Info(sqlStr)

		if _, err := dbCon.Exec(sqlStr); err != nil {
			log.Errorf("update neId[%s] compare status failed: %v", neList[i], err)
			return err
		}
	}
	log.Infof("update compare_log[%s] status success done", neList)

	return nil
}

func (proxy *mysqlDbProxy) GetCompareLog(c context.Context, pn int32, rn int32) ([]*CompareLog, error) {
	var rs []*CompareLog
	var limit int32

	dbCon, ok := proxy.dbs["mysql_online"]
	if !ok {
		return nil, fmt.Errorf("not found mysql client of %s", "mysql_online")
	}

	if pn == 1 {
		limit = 0
	} else {
		limit = rn*(pn-1) - 1
	}

	sqlStr := fmt.Sprintf(`SELECT id, ne_id, friendlyname as friendly_name, 
				task_id, create_time, end_time, status, type, item
				FROM compare_log as c INNER JOIN nodeinfo as n ON c.ne_id=n.nodeId LIMIT %d, %d`, limit, rn)
	log.Info(sqlStr)

	if err := dbCon.SelectContext(c, &rs, sqlStr); err != nil {
		log.Errorf("select all compare log information failed: %v", err)
		return nil, err
	}

	return rs, nil
}

func (proxy *mysqlDbProxy) GetCompareLogByTaskId(taskId string) ([]string, error) {
	var rs []string

	dbCon, ok := proxy.dbs["mysql_online"]
	if !ok {
		return nil, fmt.Errorf("not found mysql client of %s", "mysql_online")
	}

	sqlStr := fmt.Sprintf(`SELECT distinct(ne_id) FROM compare_log WHERE task_id="%s"`, taskId)
	log.Info(sqlStr)

	if err := dbCon.Select(&rs, sqlStr); err != nil {
		log.Errorf("select all compare log information failed: %v", err)
		return nil, err
	}

	return rs, nil
}

func (proxy *mysqlDbProxy) CountCompareLog() (total int64, err error) {
	var count int64
	dbCon, ok := proxy.dbs["mysql_online"]
	if !ok {
		return -1, fmt.Errorf("not found mysql client of %s", "mysql_online")
	}

	fail := dbCon.QueryRow("SELECT COUNT(*) FROM compare_log").Scan(&count)
	switch {
	case fail != nil:
		return -1, fmt.Errorf("can't count compare log")
	default:
		log.Info("Number of compare_log rows are %d", count)
	}

	return count, nil
}
