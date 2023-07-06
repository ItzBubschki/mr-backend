package main

import (
	"github.com/gorilla/mux"
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
	router := mux.NewRouter()

	// Register request handlers
	router.Handle("/search", searchHandler)
	router.Handle("/inspect", inspectHandler)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://www.screensociety.de", "https://screensociety.de"},
	})

	handler := c.Handler(router)

	// Start the server with CORS enabled
	log.Println("Server listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
