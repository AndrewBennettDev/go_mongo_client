package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Item struct {
	ID    string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name  string  `json:"name,omitempty" bson:"name,omitempty"`
	Price float64 `json:"price,omitempty" bson:"price,omitempty"`
}

var collection *mongo.Collection

func init() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is not set")
	}

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	databaseName := os.Getenv("MONGO_DB_NAME")
	if databaseName == "" {
		log.Fatal("MONGO_DB_NAME environment variable is not set")
	}

	collectionName := os.Getenv("MONGO_COLLECTION_NAME")
	if collectionName == "" {
		log.Fatal("MONGO_COLLECTION_NAME environment variable is not set")
	}

	collection = client.Database(databaseName).Collection(collectionName)
}

func createItem(w http.ResponseWriter, r *http.Request) {
	var newItem Item
	err := json.NewDecoder(r.Body).Decode(&newItem)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, newItem)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to create item", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result.InsertedID)
}

func getItems(w http.ResponseWriter, r *http.Request) {
	var items []Item

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to fetch items", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &items)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to fetch items", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(items)
}

func updateItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updatedItem Item
	err := json.NewDecoder(r.Body).Decode(&updatedItem)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id}
	update := bson.M{"$set": updatedItem}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to update item", http.StatusInternalServerError)
		return
	}

	if result.ModifiedCount == 0 {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result.ModifiedCount)
}

func deleteItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to delete item", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result.DeletedCount)
}
