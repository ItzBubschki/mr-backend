package main

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"flag"
	"github.com/ItzBubschki/mr-backend/main/Handlers/FirebaseHandlers"
	"github.com/ItzBubschki/mr-backend/main/Handlers/MovieHandlers"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
)

func main() {
	mongoHost := flag.String("mongoHost", "mongo", "the host of the mongo database")
	emulator := flag.Bool("emulator", false, "whether to use the firebase emulator")
	flag.Parse()
	mongoHandler, err := MovieHandlers.NewMongoHandler(*mongoHost)
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

	messagingHandler, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	searchHandler := &MovieHandlers.SearchHandler{
		Mongo: mongoHandler,
	}
	inspectHandler := &MovieHandlers.InspectHandler{
		Mongo: mongoHandler,
	}
	deletionHandler := &FirebaseHandlers.DeletionHandler{
		AuthHandler: authHandler,
		FireStore:   firestoreHandler,
	}
	restoreHandler := &FirebaseHandlers.RestoreHandler{
		AuthHandler: authHandler,
		FireStore:   firestoreHandler,
	}
	fcmHandler := &FirebaseHandlers.FcmHandler{
		AuthHandler:  authHandler,
		FireStore:    firestoreHandler,
		Messaging:    messagingHandler,
		MongoHandler: mongoHandler,
	}
	friendHandler := &FirebaseHandlers.FriendHandler{
		AuthHandler: authHandler,
		FireStore:   firestoreHandler,
		FcmHandler:  fcmHandler,
	}

	// Create a new router
	mux := http.NewServeMux()

	// Register request handlers
	mux.Handle("/search", searchHandler)
	mux.Handle("/inspect", inspectHandler)
	mux.Handle("/delete", deletionHandler)
	mux.Handle("/restore", restoreHandler)
	mux.HandleFunc("/revoke", friendHandler.RevokeRequestWrapper)
	mux.HandleFunc("/accept", friendHandler.AcceptRequestWrapper)
	mux.HandleFunc("/send", friendHandler.SendRequestWrapper)
	mux.HandleFunc("/remove", friendHandler.RemoveFriendWrapper)
	mux.HandleFunc("/decline", friendHandler.DeclineRequestWrapper)
	mux.HandleFunc("/addedToken", fcmHandler.AddedTokenWrapper)
	mux.HandleFunc("/ratedMovie", fcmHandler.RatedMovieWrapper)
	http.Handle("/", mux)

	log.Println("Server listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
