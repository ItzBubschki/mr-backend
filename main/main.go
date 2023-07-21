package main

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"flag"
	"github.com/ItzBubschki/mr-backend/main/Handlers"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
)

func main() {
	mongoHost := flag.String("mongoHost", "mongo", "the host of the mongo database")
	emulator := flag.Bool("emulator", false, "whether to use the firebase emulator")
	flag.Parse()
	mongoHandler, err := Handlers.NewMongoHandler(*mongoHost)
	if err != nil {
		log.Fatal("Failed to create MongoHandler:", err)
	}
	if *emulator {
		err := os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:9000")
		if err != nil {
			log.Fatal("Failed to set FIRESTORE_EMULATOR_HOST:", err)
		}
	}
	opt := option.WithCredentialsFile("main/serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v", err)
	}
	authHandler, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	firestoreHandler, err := app.Firestore(context.Background())
	if err != nil {
		log.Fatalf("error getting Firestore client: %v\n", err)
	}

	searchHandler := &Handlers.SearchHandler{
		Mongo: mongoHandler,
	}
	inspectHandler := &Handlers.InspectHandler{
		Mongo: mongoHandler,
	}
	deletionHandler := &Handlers.DeletionHandler{
		AuthHandler: authHandler,
		FireStore:   firestoreHandler,
	}
	restoreHandler := &Handlers.RestoreHandler{
		AuthHandler: authHandler,
		FireStore:   firestoreHandler,
	}

	// Create a new router
	mux := http.NewServeMux()

	// Register request handlers
	mux.Handle("/search", searchHandler)
	mux.Handle("/inspect", inspectHandler)
	mux.Handle("/delete", deletionHandler)
	mux.Handle("/restore", restoreHandler)
	http.Handle("/", mux)

	log.Println("Server listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
