package FirebaseHandlers

import (
	"cloud.google.com/go/firestore"
	"context"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
	"log"
	"net/http"
)

type RestoreHandler struct {
	AuthHandler *auth.Client
	FireStore   *firestore.Client
}

func (rh *RestoreHandler) retrieveOldUserData(email string) (string, []string) {
	docs, err := rh.FireStore.Collection("ArchivedUsers").Where("email", "==", email).Documents(context.Background()).GetAll()
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return "", nil
	}
	if len(docs) == 0 {
		return "", nil
	}
	var user User
	err = docs[0].DataTo(&user)
	if err != nil {
		log.Printf("Failed to convert data: %v", err)
		return "", nil
	}
	return docs[0].Ref.ID, user.Friends
}

func (rh *RestoreHandler) restoreUserData(newUserId, oldUserId string) {
	userDoc := rh.FireStore.Collection("ArchivedUsers").Doc(oldUserId)
	userData, err := userDoc.Get(context.Background())
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	_, err = rh.FireStore.Collection("Users").Doc(newUserId).Set(context.Background(), userData.Data(), firestore.MergeAll)
	if err != nil {
		log.Printf("Failed to restore user: %v", err)
		return
	}
	_, err = userDoc.Delete(context.Background())
	return
}

func (rh *RestoreHandler) restoreRatings(newUserId, oldUserId string) {
	query := rh.FireStore.Collection("ArchivedRatings").Where("userId", "==", oldUserId)
	ratingsCollection := rh.FireStore.Collection("Ratings")
	err := rh.FireStore.RunTransaction(context.Background(), func(ctx context.Context, tx *firestore.Transaction) error {
		iter := tx.Documents(query)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to iterate: %v", err)
				return err
			}
			var rating Rating
			err = doc.DataTo(&rating)
			if err != nil {
				log.Printf("Failed to convert data: %v", err)
				return err
			}
			log.Printf("Restoring rating: %v", doc.Ref.ID)
			rating.UserId = newUserId
			_, err = ratingsCollection.Doc(doc.Ref.ID).Set(ctx, rating)
			if err != nil {
				log.Printf("Failed to restore rating: %v", err)
				return err
			}
			err = tx.Delete(doc.Ref)
			if err != nil {
				log.Printf("Failed to delete rating: %v", err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to restore ratings: %v", err)
		return
	}
}

func (rh *RestoreHandler) restoreUserFriends(newUserId string, friends []string) {
	for _, friend := range friends {
		doc := rh.FireStore.Collection("Users").Doc(friend)
		_, err := doc.Update(context.Background(), []firestore.Update{
			{
				Path:  "friends",
				Value: firestore.ArrayUnion(newUserId),
			},
		})
		if err != nil {
			log.Printf("Failed to restore friend: %v", err)
		}
	}
}

func (rh *RestoreHandler) checkIfRestoreAvailable(email string) bool {
	id, _ := rh.retrieveOldUserData(email)
	return id != ""
}

func (rh *RestoreHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Printf("Failed to write response: %v", err)
		}
		return
	} else if r.Method == http.MethodGet {
		available := rh.checkIfRestoreAvailable(r.URL.Query().Get("email"))
		if available {
			_, err := w.Write([]byte("OK"))
			if err != nil {
				log.Printf("Failed to write response: %v", err)
			}
			return
		} else {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Not found"))
			if err != nil {
				log.Printf("Failed to write response: %v", err)
			}
			return
		}
	} else if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//get the jwt auth token from the header
	idToken := r.Header.Get("Authorization")
	if idToken == "" {
		log.Println("No token found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token, err := rh.AuthHandler.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		log.Printf("error verifying ID token: %v\n", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	if token.Claims["email_verified"] != true {
		log.Println("Email not verified")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	oldId, friends := rh.retrieveOldUserData(token.Claims["email"].(string))
	if oldId == "" {
		log.Println("No archived user found")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	go rh.restoreUserData(token.UID, oldId)
	go rh.restoreRatings(token.UID, oldId)
	go rh.restoreUserFriends(token.UID, friends)

	//respond with 200 OK
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("OK"))
	if err != nil {
		return
	}
}
