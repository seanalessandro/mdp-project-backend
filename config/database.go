package config

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

func ConnectDB() {
	// MongoDB connection string - replace with your actual connection string
	mongoURI := "mongodb://localhost:27017" // Change this to your MongoDB URI
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	log.Println("Connected to MongoDB successfully!")
	
	// Set the database - replace "mdp_project" with your database name
	DB = client.Database("mdp_project")
}

func GetCollection(collectionName string) *mongo.Collection {
	return DB.Collection(collectionName)
}
