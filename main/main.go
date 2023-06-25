package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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
	router := mux.NewRouter()

	// Register request handlers
	router.Handle("/search", searchHandler)
	router.Handle("/inspect", inspectHandler)

	// Enable CORS
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type"})
	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:19000"}) // Update with your frontend origin
	allowedMethods := handlers.AllowedMethods([]string{"GET"})

	// Start the server with CORS enabled
	log.Println("Server listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods)(router)))
}
