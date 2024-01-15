# tianyuan
Network resource consistency inspection and reconciliation system

## 项目介绍

当前网管系统在长期运行阶段，暴露许多数据资源不一致问题（控制器内部以及设备侧数据），这些问题有大有小，严重的影响网络建设快速扩容带宽需求的业务下发成功率，
以及对Controller管控设备产生影响，主要包括：

<img width="" src="/uploads/132A9AA819F346EFAB6057C5B2A5D4A7/image.png" alt="image.png" />

## 架构设计
基于Tencent的开源trpc-go框架搭建服务
<img width="" src="/uploads/FC1289F31D1E496B9F4DAC663E3B1672/image.png" alt="image.png" />

## rcc_server在线架构
<img width="" src="/uploads/BD25C952BB9D4023BFC4D8215C456807/image.png" alt="image.png" />

## 主要功能
### 技术点：
#### 1	定制化巡检服务
<img width="" src="/uploads/6434A032041046D78C6488669DC28132/image.png" alt="image.png" />


* 支持单Ne巡检+配置下发
* 支持全部设备定期定时巡检
* 支持指定批量设备定时巡检

#### 2	解耦服务

* 增加Ral资源服务访问层
* 支持websocket上报
* 支持算子服务隔离计算
* 支持数据流式计算
* 支持控制器联动处理
* 锁机制

#### 3	灵活支持下游产出

* 单设备下发配置后查询状态
* 进度条结果反馈
* 巡检结果分析算法
* 汇总结果报告 邮件 企业微信rtx通知


## 如何使用


### 部署环境

TKE集群

### 项目编译

```shell
go build -o rcc main.go
```

### 项目启动

```shell
./rcc -conf ./conf/rcc.yaml 
```

### API测试
```shell
curl --location --request POST 'http://【host】/trpc.tianyuan.rcc_server.Rcc/CompareNe' \
--header 'Content-Type: application/json' \
--data-raw '{
    "ne_id": [
        "Site-xxx#Ne-xxx"
    ]
}'
```
