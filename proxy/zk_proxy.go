package proxy

import (
	"encoding/json"
	"errors"
	"rcc/mod"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"github.com/go-zookeeper/zk"
)

// 向zk获取控制器地址
func getServiceAddr(name string) (string, error) {
	conn, _, err := zk.Connect(conf.zkAddr, conf.timeout*time.Second)
	if err != nil {
		log.Errorf("conn zk failed(%v)", err)
		return "", err
	}
	defer conn.Close()

	children, err := getChildren(conn, conf.zkPath+"/"+name)
	if err != nil {
		log.Errorf("get children failed(%v)", err)
		return "", err
	}

	for _, v := range children {
		data, err := get(conn, conf.zkPath+"/"+name+"/"+v)
		if err != nil {
			log.Errorf("get children data failed(%v)", err)
			continue
		}
		log.Debugf("detail: %s", string(data[:]))
		detail := &mod.InstanceDetails{}
		err = json.Unmarshal(data, detail)
		if err != nil {
			log.Errorf("unmarshal detail failed(%v)", err)
			continue
		}
		if detail.Namespace == conf.namespace && detail.MyIp != "" {
			return detail.MyIp, nil
		}
	}

	return "", errors.New("not found service addr")
}

// 增
// flags有4种取值：
// 0:永久，除非手动删除
// zk.FlagEphemeral = 1:短暂，session断开则该节点也被删除
// zk.FlagSequence  = 2:会自动在节点后面添加序号
// 3:Ephemeral和Sequence，即，短暂且自动添加序号
func add(conn *zk.Conn, path string, data []byte, flags int32) error {
	// 获取访问控制权限
	acls := zk.WorldACL(zk.PermAll)
	_, err := conn.Create(path, data, flags, acls)
	return err
}

// 查
func get(conn *zk.Conn, path string) ([]byte, error) {
	// log.Infof("path: %s", path)
	data, _, err := conn.Get(path)
	return data, err
}

// 查children
func getChildren(conn *zk.Conn, path string) ([]string, error) {
	children, _, err := conn.Children(path)
	return children, err
}

// 改
// 删改与增不同在于其函数中的version参数,其中version是用于 CAS支持
// 可以通过此种方式保证原子性
func modify(conn *zk.Conn, path string, data []byte) error {
	_, sate, err := conn.Get(path)
	if err != nil {
		return err
	}
	_, err = conn.Set(path, data, sate.Version)
	return err
}

// 删
func del(conn *zk.Conn, path string) error {
	_, sate, err := conn.Get(path)
	if err != nil {
		return err
	}
	err = conn.Delete(path, sate.Version)
	return err
}
