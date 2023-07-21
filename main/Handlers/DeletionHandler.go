package Handlers

import (
	"cloud.google.com/go/firestore"
	"context"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
	"log"
	"net/http"
	"time"
)

type DeletionHandler struct {
	AuthHandler *auth.Client
	FireStore   *firestore.Client
}

type Rating struct {
	UserId    string    `firestore:"userId"`
	MovieId   string    `firestore:"movieId"`
	Rating    float64   `firestore:"rating"`
	Comment   string    `firestore:"comment"`
	Timestamp time.Time `firestore:"timestamp"`
	ExpiresAt time.Time `firestore:"expiresAt,omitempty"`
}

type User struct {
	Email          string    `firestore:"email"`
	Friends        []string  `firestore:"friends,omitempty"`
	Name           string    `firestore:"name"`
	Picture        string    `firestore:"picture"`
	RatedMovies    []string  `firestore:"ratedMovies,omitempty"`
	FriendRequests []string  `firestore:"friendRequests,omitempty"`
	ExpiresAt      time.Time `firestore:"expiresAt,omitempty"`
}

func (d *DeletionHandler) moveUserRatings(userId string) {
	query := d.FireStore.Collection("Ratings").Where("userId", "==", userId)
	archivedRatings := d.FireStore.Collection("ArchivedRatings")
	err := d.FireStore.RunTransaction(context.Background(), func(ctx context.Context, tx *firestore.Transaction) error {
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
			rating.ExpiresAt = time.Now().Add(time.Hour * 24 * 14)
			newRating := archivedRatings.Doc(doc.Ref.ID)
			err = tx.Create(newRating, rating)
			if err != nil {
				log.Printf("Failed to create: %v", err)
				return err
			}
			err = tx.Delete(doc.Ref)
			if err != nil {
				log.Printf("Failed to delete: %v", err)
				return err
			}
		}
		return nil
	})
	if err != nil {
		return
	}
}

func (d *DeletionHandler) moveUserData(userId string) {
	userDoc := d.FireStore.Collection("Users").Doc(userId)
	userData, err := userDoc.Get(context.Background())
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}
	var user User
	err = userData.DataTo(&user)
	if err != nil {
		log.Printf("Failed to convert data: %v", err)
		return
	}
	user.ExpiresAt = time.Now().Add(time.Hour * 24 * 14)
	_, err = d.FireStore.Collection("ArchivedUsers").Doc(userId).Set(context.Background(), user)
	if err != nil {
		log.Printf("Failed to archive user: %v", err)
		return
	}
	_, err = userDoc.Delete(context.Background())
}

func (d *DeletionHandler) removeUserFromFriends(userId string) {
	query := d.FireStore.Collection("Users").Where("friends", "array-contains", userId)
	d.removeUserFromFieldInQuery(query, "friends", userId)

	query = d.FireStore.Collection("Users").Where("friendRequests", "array-contains", userId)
	d.removeUserFromFieldInQuery(query, "friendRequests", userId)
}

func (d *DeletionHandler) removeUserFromFieldInQuery(query firestore.Query, field string, userId string) {
	err := d.FireStore.RunTransaction(context.Background(), func(ctx context.Context, tx *firestore.Transaction) error {
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
			var user User
			err = doc.DataTo(&user)
			if err != nil {
				log.Printf("Failed to convert data: %v", err)
				return err
			}
			log.Printf("Removing user %v from %s of %v", userId, field, doc.Ref.ID)
			err = tx.Update(doc.Ref, []firestore.Update{
				{
					Path:  field,
					Value: firestore.ArrayRemove(userId),
				},
			})
		}
		return nil
	})
	if err != nil {
		log.Printf("Failed to remove user from friends: %v", err)
	}
}

func (d *DeletionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	token, err := d.AuthHandler.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		log.Printf("error verifying ID token: %v\n", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	log.Printf("Verified ID token: %v\n", token)

	d.moveUserRatings(token.UID)
	d.moveUserData(token.UID)
	d.removeUserFromFriends(token.UID)
	//respond with 200 OK
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("OK"))
	if err != nil {
		return
	}
}
