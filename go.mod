module github.com/DoubleZ0405/tianyuaxn

go 1.18

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/go-zookeeper/zk v1.0.2
	github.com/gorilla/websocket v1.4.2
	github.com/imdario/mergo v0.3.12
	github.com/jmoiron/sqlx v1.3.5
	github.com/r3labs/diff v1.1.0
	github.com/satori/go.uuid v1.2.0
	go.mongodb.org/mongo-driver v1.12.1
	gopkg.in/yaml.v3 v3.0.1
	trpc.group/trpc-go/trpc-go v1.0.2
)

require github.com/DoubleZ0405/tianyuan v0.0.0-20240115032629-252185d97551 // indirect

replace google.golang.org/protobuf v1.26.0 => google.golang.org/protobuf v1.25.0
