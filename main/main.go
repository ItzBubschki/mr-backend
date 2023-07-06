package main

import (
	"github.com/rs/cors"
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

	// Create a new router
	mux := http.NewServeMux()

	// Register request handlers
	mux.Handle("/search", searchHandler)
	mux.Handle("/inspect", inspectHandler)

	handler := cors.Default().Handler(mux)

	// Start the server with CORS enabled
	log.Println("Server listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
