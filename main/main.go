package main

import (
	"log"
	"mr-backend/main/Handlers"
	"net/http"
)

func main() {
	mongoHandler, err := Handlers.NewMongoHandler()
	if err != nil {
		log.Fatal("Failed to create MongoHandler:", err)
	}
	searchHandler := &Handlers.SearchHandler{
		Mongo: mongoHandler,
	}
	inspectHandler := &Handlers.InspectHandler{
		Mongo: mongoHandler,
	}
	// Register request handlers
	http.Handle("/search", searchHandler)
	http.Handle("/inspect", inspectHandler)

	// Start the server
	log.Println("Server listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
