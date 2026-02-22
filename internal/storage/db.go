package storage

import (
	"os"
	"log"
	"time"
	"fmt"
	"context"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var Client *mongo.Client

func ConnectDB() *mongo.Client{
	
	uri := os.Getenv("MONGODB_URL")
	
	if uri == "" {
		log.Fatal("Failed to get mongodb url from env.")
	}

	clientOptions := options.Client().ApplyURI(uri).SetMaxPoolSize(100).SetMinPoolSize(10).SetMaxConnIdleTime(30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(clientOptions)

	if err != nil {
		log.Fatal("Failed to connect to MongoDB: ", err)
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		log.Fatal("Could not ping MongoDb: ", err)
	}

	fmt.Println("Connected to MongoDb and Connection Pool warmed up!")
	Client = client
	return client
}

func GetCollection(collectionName string) *mongo.Collection {
	dbName := os.Getenv("DB_NAME")
	return Client.Database(dbName).Collection(collectionName)
}
