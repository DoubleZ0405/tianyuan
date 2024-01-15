package db

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

// COLL mongo collection集合名
type COLL int

const (
	INSPECTION COLL = 0
)

// String COLL转成对应的string
func (c COLL) String() string {
	switch c {
	case INSPECTION:
		return "inspection"
	default:
		return "invalid"
	}
}

// MdbProxy  是Db的接口
type MdbProxy interface {
	//Opts(opts ...client.Option)

	//Do(ctx context.Context, cmd string, db string, coll string, args map[string]interface{}) (interface{}, error)

	InsertOne(ctx context.Context, coll COLL, document interface{}) (id string, err error)

	InsertMany(ctx context.Context, coll COLL, documents []interface{}) (ids []string, err error)

	UpdateOne(ctx context.Context, coll COLL, id string, document interface{}) error

	FindOneAndUpdate(ctx context.Context, coll COLL, filter, document interface{}) error

	DeleteOne(ctx context.Context, coll COLL, id string) error

	FindOneAndDelete(ctx context.Context, coll COLL, filter interface{}) error

	FindManyAndDelete(ctx context.Context, coll COLL, filter interface{}) error

	GetOne(ctx context.Context, coll COLL, id string, result interface{}) (exist bool, err error)

	Exist(ctx context.Context, coll COLL, filter interface{}, excludeKeys ...string) (bool, error)

	Find(ctx context.Context, coll COLL, filter, result interface{}) error
}

// MongoDBClient 封装 MongoDB 客户端
type MongoDBClient struct {
	dbname string
	client *mongo.Client
}

// NewMongoDBClient 是初始化mongoDB
func NewMongoDBClient(dbName, uri string) MdbProxy {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		panic(err)
	}
	cancel()
	return &MongoDBClient{
		dbname: dbName,
		client: client,
	}
}

// InsertOne 插入单条数据
func (m *MongoDBClient) InsertOne(ctx context.Context, coll COLL, document interface{}) (string, error) {
	data, err := buildDocument(document)
	if err != nil {
		return "", err
	}
	_, _ = m.InsertOne(ctx, coll, data)
	if err != nil {
		return "", err
	}

	//if result, ok := got.(map[string]interface{}); ok {
	//	if id, ok := result["InsertedID"].(string); ok {
	//		return id, err
	//	}
	//}
	return "", fmt.Errorf("insert one failed(%v)", document)
}

// 插入多条数据
func (m *MongoDBClient) InsertMany(ctx context.Context, coll COLL, documents []interface{}) ([]string, error) {
	data, err := buildDocument(documents)
	if err != nil {
		return nil, err
	}
	_, _ = m.InsertMany(ctx, coll, data)
	if err != nil {
		return nil, err
	}
	//result, ok := got.(map[string]interface{})
	//if ok {
	//	if insertedIDs, ok := result["InsertedIDs"].([]interface{}); ok {
	//		var ids []string
	//		for _, insertedID := range insertedIDs {
	//			if id, ok := insertedID.(string); ok {
	//				ids = append(ids, id)
	//			}
	//		}
	//		return ids, err
	//	}
	//}
	return nil, fmt.Errorf("insert many failed(%v)", documents)
}

// 基于ID更新一条数据
func (m *MongoDBClient) UpdateOne(ctx context.Context, coll COLL, id string, document interface{}) error {
	data, err := buildIdFilter(id)
	if err != nil {
		return err
	}
	data, err = buildUpdate(data, document)
	if err != nil {
		return err
	}
	//_, _ = m.proxy.Do(ctx, mongodb.UpdateOne, m.dbName, coll.String(), data)
	return err
}

// 匹配一条数据并更新
func (m *MongoDBClient) FindOneAndUpdate(ctx context.Context, coll COLL, filter, document interface{}) error {
	data, err := buildFilter(filter)
	if err != nil {
		return err
	}
	data, err = buildUpdate(data, document)
	if err != nil {
		return err
	}
	//_, err = m.proxy.Do(ctx, mongodb.UpdateOne, m.dbName, coll.String(), data)
	return err
}

// 基于ID删除一条数据
func (m *MongoDBClient) DeleteOne(ctx context.Context, coll COLL, id string) error {
	_, err := buildIdFilter(id)
	if err != nil {
		return err
	}
	//_, err = m.proxy.Do(ctx, mongodb.DeleteOne, m.dbName, coll.String(), filter)
	return err
}

// 匹配一条数据并更新
func (m *MongoDBClient) FindOneAndDelete(ctx context.Context, coll COLL, filter interface{}) error {
	if ok, err := m.Exist(ctx, coll, filter); err != nil || !ok {
		return err
	}
	_, err := buildFilter(filter)
	if err != nil {
		return err
	}
	//_, err = m.proxy.Do(ctx, mongodb.FindOneAndDelete, m.dbName, coll.String(), data)
	return err
}

// 匹配多条数据并更新
func (m *MongoDBClient) FindManyAndDelete(ctx context.Context, coll COLL, filter interface{}) error {
	if ok, err := m.Exist(ctx, coll, filter); err != nil || !ok {
		return err
	}
	_, err := buildFilter(filter)
	if err != nil {
		return err
	}
	//_, err = m.proxy.Do(ctx, mongodb.DeleteMany, m.dbName, coll.String(), data)
	return err
}

// 基于ID查询一条数据
func (m *MongoDBClient) GetOne(ctx context.Context, coll COLL, id string, result interface{}) (bool, error) {
	_, err := buildIdFilter(id)
	if err != nil {
		return false, err
	}
	//_, err = toBytes(m.proxy.Do(ctx, mongodb.Find, m.dbName, coll.String(), filter))
	//if err != nil || len(bytes) == 0 {
	//	return false, err
	//}
	return true, json.Unmarshal(nil, &result)
}

// 判断是否存在匹配数据
func (m *MongoDBClient) Exist(ctx context.Context, coll COLL, filter interface{}, excludeKeys ...string) (bool, error) {
	data, err := buildFilter(filter, excludeKeys...)
	if err != nil {
		return false, err
	}
	//result, err := m.proxy.Do(ctx, mongodb.Find, m.dbName, coll.String(), data)
	if err != nil {
		return false, err
	}
	//if slices, ok := result.([]map[string]interface{}); ok {
	//	return len(slices) != 0, nil
	//}
	return false, fmt.Errorf("result(%v) is not map[string]interface{}", data)
}

// 查找匹配数据
func (m *MongoDBClient) Find(ctx context.Context, coll COLL, filter, result interface{}) error {
	_, err := buildFilter(filter)
	if err != nil {
		return err
	}
	//bytes, err := toBytesSlice(m.proxy.Do(ctx, mongodb.Find, m.dbName, coll.String(), data))
	//if err != nil || len(bytes) == 0 {
	//	return err
	//}
	return json.Unmarshal(nil, &result)
}

// 构建document
func buildDocument(data interface{}) ([]interface{}, error) {
	switch reply := data.(type) {
	case []interface{}:
		var document []interface{}
		for _, r := range reply {
			mapStruct, err := toMap(r)
			if err != nil {
				return nil, err
			}
			delete(mapStruct, "id")
			document = append(document, mapStruct)
		}
		return []interface{}{
			document,
		}, nil
	default:
		mapStruct, err := toMap(data)
		if err != nil {
			return nil, err
		}
		delete(mapStruct, "id")
		return []interface{}{
			mapStruct,
		}, nil
	}
}

// 构建update
func buildUpdate(filter map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	mapStruct, err := toMap(data)
	if err != nil {
		return nil, err
	}
	delete(mapStruct, "id")
	filter["update"] = map[string]interface{}{
		"$set": mapStruct,
	}
	return filter, nil
}

// 构建基于ID的filter
func buildIdFilter(id string) (map[string]interface{}, error) {
	var filter map[string]interface{} = make(map[string]interface{})
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		oid = primitive.NilObjectID
	}
	filter["_id"] = oid
	return map[string]interface{}{
		"filter": filter,
	}, nil
}

// 构建filter
func buildFilter(data interface{}, excludeKeys ...string) (map[string]interface{}, error) {
	mapStruct, err := toMap(data)
	if err != nil {
		return nil, err
	}
	if val, ok := mapStruct["id"]; ok {
		if idStr, ok := val.(string); ok {
			oid, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				return nil, err
			}
			mapStruct["_id"] = oid
			delete(mapStruct, "id")
		}
	} else if len(excludeKeys) > 0 {
		var ids []primitive.ObjectID
		for _, idStr := range excludeKeys {
			oid, err := primitive.ObjectIDFromHex(idStr)
			if err != nil {
				return nil, err
			}
			ids = append(ids, oid)
		}
		mapStruct["_id"] = map[string][]primitive.ObjectID{
			"$nin": ids,
		}
	}
	return map[string]interface{}{
		"filter": mapStruct,
	}, nil
}

// 将结构转成map
func toMap(data interface{}) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var mapStruct map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &mapStruct); err != nil {
		return nil, err
	}
	return mapStruct, nil
}

// 将结构转成bytes
func toBytes(data interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if slices, ok := data.([]map[string]interface{}); ok {
		if len(slices) == 0 {
			return []byte{}, err
		}
		for _, slice := range slices {
			if val, ok := slice["_id"]; ok {
				if oid, ok := val.(primitive.ObjectID); ok {
					slice["id"] = oid.Hex()
				}
			}
			return json.Marshal(slice)
		}
	}
	return nil, fmt.Errorf("data(%v) is not map[string]interface{}", data)
}

// 将slice结构转成bytes
func toBytesSlice(data interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if slices, ok := data.([]map[string]interface{}); ok {
		if len(slices) == 0 {
			return []byte{}, err
		}
		var jsonSlice []json.RawMessage
		for _, slice := range slices {
			if val, ok := slice["_id"]; ok {
				if oid, ok := val.(primitive.ObjectID); ok {
					slice["id"] = oid.Hex()
				}
			}
			bytes, err := json.Marshal(slice)
			if err != nil {
				return bytes, err
			}
			jsonSlice = append(jsonSlice, bytes)
		}
		return json.Marshal(jsonSlice)
	}
	return nil, fmt.Errorf("data(%v) is not map[string]interface{}", data)
}
