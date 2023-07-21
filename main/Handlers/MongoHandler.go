package Handlers

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type MongoHandler struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoHandler(mongoHost string) (*MongoHandler, error) {
	//use a flag for the mongo host

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:27017", mongoHost))
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	// Set up collection
	collection := client.Database("mr-cache").Collection("movies")

	// Create and return MongoHandler instance
	handler := &MongoHandler{
		client:     client,
		collection: collection,
	}
	return handler, nil
}

func (m *MongoHandler) FetchFromCache(movieId string) (MovieResponse, error) {
	filter := bson.M{"imdbid": movieId}
	result := m.collection.FindOne(context.Background(), filter)
	if err := result.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return MovieResponse{}, nil // Document not found in cache
		}
		return MovieResponse{}, err // Error occurred while fetching from cache
	}

	var movie MovieResponse
	if err := result.Decode(&movie); err != nil {
		return MovieResponse{}, err // Error decoding cache value
	}

	return movie, nil
}

func (m *MongoHandler) SaveInCache(movies []MovieResponse) {
	for _, movie := range movies {
		// Check if a movie with the same imdbId already exists in the database
		filter := bson.M{"imdbid": movie.IMDBID}
		existingMovie := m.collection.FindOne(context.Background(), filter)
		if existingMovie.Err() == nil {
			// A movie with the same imdbId already exists, skip saving
			continue
		}

		// No existing movie found, save the current movie
		_, err := m.collection.InsertOne(context.Background(), movie)
		if err != nil {
			log.Println("Failed to save cache:", err)
			// Handle the error accordingly
		}
	}
}
