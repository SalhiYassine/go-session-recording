package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	mongoURI          = "mongodb://localhost:27027"
	dbName            = "mydb"
	sessionCollection = "sessions"
	eventCollection   = "events"
)

type Session struct {
	ID                primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ClientID          primitive.ObjectID `json:"clientId,omitempty" bson:"clientId,omitempty"`
	VisitorID         primitive.ObjectID `json:"visitorId,omitempty" bson:"visitorId,omitempty"`
	LastEventTime     time.Time          `json:"lastEventTime" bson:"lastEventTime,omitempty"`
	DurationInSeconds int                `json:"durationInSeconds" bson:"durationInSeconds,omitempty"`
	CreatedAt         time.Time          `json:"createdAt" bson:"createdAt,omitempty"`
	UpdatedAt         time.Time          `json:"updatedAt" bson:"updatedAt,omitempty"`
}

type Event struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	SessionID primitive.ObjectID `json:"sessionId,omitempty" bson:"sessionId,omitempty"`

	DomEvent  string    `json:"domEvent" bson:"domEvent"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt,omitempty"`
}

func main() {

	// Connect to MongoDB.
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Ping MongoDB to verify the connection.
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	// Set up Gin router.
	r := gin.Default()

	// Define routes.
	r.GET("/sessions", listSessionsHandler(client))
	r.POST("/sessions", createSessionHandler(client))
	r.POST("/sessions/:sessionId/event", createEventHandler(client))

	// Start server.
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// listSessionsHandler retrieves sessions filtered by client and visitor IDs.
func listSessionsHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		offset, err := strconv.Atoi(c.Query("offset"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Offset does not match the format exepected, expected a number."})
			return
		}

		limit, err := strconv.Atoi(c.Query("limit"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Limit does not match the format expected, expected a number."})
			return
		}

		clientId := c.Query("clientId")
		visitorId := c.Query("visitorId")

		clientObjectId, err := primitive.ObjectIDFromHex(clientId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Client ID is malformed, does not conform to hex format."})
			return
		}

		filter := bson.M{"clientId": clientObjectId}

		if visitorId != "" {

			visitorObjectId, err := primitive.ObjectIDFromHex(clientId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Visitor ID is malformed, does not conform to hex format."})
				return
			}

			filter["visitorId"] = visitorObjectId
		}

		var sessions []Session

		cursor, err := client.Database(dbName).Collection(sessionCollection).Find(ctx, filter, options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve from db."})
			return
		}

		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var session Session
			if err := cursor.Decode(&session); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding sessions from db."})
				return
			}

			sessions = append(sessions, session)
		}

		if err := cursor.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong when retrieving the sessions."})
			return
		}

		c.JSON(http.StatusOK, sessions)
	}
}

func createSessionHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse request body.
		var params Session
		if err := c.ShouldBindJSON(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if params.ClientID == primitive.NilObjectID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Client Id provided."})
			return
		}

		if params.VisitorID == primitive.NilObjectID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Visitor Id provided."})
			return
		}

		now := time.Now()
		session := Session{
			ClientID:  params.ClientID,
			VisitorID: params.VisitorID,

			DurationInSeconds: 0,

			LastEventTime: now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sessionInsertResult, err := client.Database(dbName).Collection(sessionCollection).InsertOne(ctx, session)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		session.ID = sessionInsertResult.InsertedID.(primitive.ObjectID)
		c.JSON(http.StatusCreated, session)
	}
}

func createEventHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		sessionId := c.Param("sessionId")

		sessionObjectId, err := primitive.ObjectIDFromHex(sessionId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Client ID is malformed, does not conform to hex format."})
			return
		}
		var params Event
		if err := c.ShouldBindJSON(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if params.DomEvent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dom event needs to be provided."})
			return
		}

		now := time.Now()
		Event := Event{
			SessionID: sessionObjectId,
			DomEvent:  params.DomEvent,
			CreatedAt: now,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		eventInsertResult, err := client.Database(dbName).Collection(eventCollection).InsertOne(ctx, Event)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		Event.ID = eventInsertResult.InsertedID.(primitive.ObjectID)

		c.JSON(http.StatusCreated, Event)
	}
}
