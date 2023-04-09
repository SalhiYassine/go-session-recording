package repository

import (
	"context"
	"time"

	"github.com/SalhiYassine/go-session-recording/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoClient struct {
	collection *mongo.Collection
}

func getMongoCollection(collectionName string, dbUri string, database string) *mongo.Collection {
	client, err := mongo.NewClient(options.Client().ApplyURI(dbUri))

	if err != nil {
		panic(err)
	}

	err = client.Connect(context.Background())

	if err != nil {
		panic(err)
	}

	return client.Database(database).Collection(collectionName)
}

func NewSessionRepository(db string, dbURi string) session.SessionRepository {
	collection := getMongoCollection("sessions", "mongodb://localhost:27017", db)
	return &mongoClient{collection}
}

func (r *mongoClient) FindById(id string) (*session.Session, error) {
	filter := bson.M{"_id": id}

	var result session.Session
	err := r.collection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *mongoClient) FindAllByClientId(clientId string) ([]*session.Session, error) {
	filter := bson.M{"clientId": clientId}

	cur, err := r.collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}

	defer cur.Close(context.Background())

	var sessions []*session.Session
	for cur.Next(context.Background()) {
		var session session.Session
		err := cur.Decode(&session)
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, &session)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

type sessionCreationParamsBoilerplate struct {
	ClientId          string
	VisitorId         string
	LastEventTime     time.Time
	DurationInSeconds int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r *mongoClient) Store(session *session.SessionCreationParams) error {
	formattedSession := sessionCreationParamsBoilerplate{
		ClientId:          session.ClientId,
		VisitorId:         session.VisitorId,
		LastEventTime:     time.Now(),
		DurationInSeconds: 0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	_, err := r.collection.InsertOne(context.Background(), formattedSession)
	return err
}

func (r *mongoClient) Update(session *session.Session) error {
	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}

	_, err := r.collection.UpdateOne(context.Background(), filter, update)
	return err
}

func (r *mongoClient) Delete(id string) error {
	filter := bson.M{"_id": id}

	_, err := r.collection.DeleteOne(context.Background(), filter)
	return err
}
