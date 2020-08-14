package utils

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IndexList struct {
	coll     *mongo.Collection
	nameList []string
}

func NewIndexList(coll *mongo.Collection) (*IndexList, error) {
	cursor, err := coll.Indexes().List(context.Background(), options.ListIndexes())
	if err != nil {
		return nil, err
	}

	list := IndexList{
		coll:     coll,
		nameList: make([]string, 0),
	}

	defer func() { _ = cursor.Close(context.Background()) }()

	for cursor.Next(context.Background()) {
		value := cursor.Current.Lookup("name")
		if value.Type == bsontype.String {
			name := value.StringValue()
			list.nameList = append(list.nameList, name)
		}
	}

	return &list, nil
}

func (list *IndexList) EnsureIndex(name string, unique bool, model mongo.IndexModel) error {
	for _, item := range list.nameList {
		if item == name {
			// Already exists
			return nil
		}
	}

	model.Options = options.Index().SetUnique(unique).SetName(name)

	name, err := list.coll.Indexes().CreateOne(context.Background(), model, options.CreateIndexes())

	if err != nil {
		return err
	}

	return nil
}

func (list *IndexList) EnsureSimpleIndex(key string, unique bool) error {
	return list.EnsureIndex(key, unique, mongo.IndexModel{
		Keys: bson.M{key: 1},
	})
}
