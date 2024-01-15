package proxy

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"trpc.group/trpc-go/trpc-go/config"

	"trpc.group/trpc-go/trpc-go/log"
)

// 代理配置
type proxyConf struct {
	namespace    string
	zkAddr       []string
	zkPath       string
	ctrlName     string
	ctrlUserName string
	ctrlPwd      string
	timeout      time.Duration
}

var conf *proxyConf

// SetConfig 设置代理的配置
func SetConfig(cfg config.Config) {
	conf = &proxyConf{
		namespace:    cfg.GetString("client.proxy.namespace", ""),
		ctrlName:     cfg.GetString("client.proxy.controller.name", "controller"),
		ctrlUserName: cfg.GetString("client.proxy.controller.username", ""),
		ctrlPwd:      cfg.GetString("client.proxy.controller.password", ""),
		timeout:      time.Duration(cfg.GetInt("client.proxy.timeout", 10000)),
	}

	zkUrl := cfg.GetString("client.proxy.zk", "")
	idx := strings.Index(zkUrl, "/")
	if idx <= 0 {
		conf.zkAddr = []string{}
		conf.zkPath = zkUrl
	} else {
		conf.zkAddr = strings.Split(zkUrl[:idx], ",")
		conf.zkPath = zkUrl[idx:]
	}
	log.Infof("proxy config: namespace(%s), controller_name(%s), zk_addr(%v), zk_path(%s)",
		conf.namespace, conf.ctrlName, conf.zkAddr, conf.zkPath)
}

// 执行POST请求
func doPost(url, name, pwd string, reqBody []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(string(reqBody)))
	if name != "" && pwd != "" {
		req.SetBasicAuth(name, pwd)
	}
	req.Header.Set("Content-Type", "application/json")
	//ctx, cancel := context.WithTimeout(context.Background(), conf.timeout*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()
	req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

// 执行GET请求
func doGet(url, name, pwd string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if name != "" && pwd != "" {
		req.SetBasicAuth(name, pwd)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5)
	defer cancel()
	req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}
