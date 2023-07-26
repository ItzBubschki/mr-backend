package FirebaseHandlers

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"fmt"
	"github.com/ItzBubschki/mr-backend/main/Handlers"
	"github.com/ItzBubschki/mr-backend/main/Handlers/MovieHandlers"
	"log"
	"net/http"
	"sync"
	"time"
)

type FcmHandler struct {
	AuthHandler  *auth.Client
	FireStore    *firestore.Client
	Messaging    *messaging.Client
	MongoHandler *MovieHandlers.MongoHandler
	mutex        sync.Mutex
	userRatings  map[string]RatingEvent
}

type RatingEvent struct {
	UserID   string
	MovieID  string
	DateTime time.Time
	Multiple bool
}

type MessageData struct {
	Link string
}

func (fcm *FcmHandler) getUserInfo(userId string) User {
	user, err := fcm.FireStore.Collection("Users").Doc(userId).Get(context.Background())
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return User{}
	}
	var userData User
	err = user.DataTo(&userData)
	if err != nil {
		log.Printf("Failed to convert data: %v", err)
		return User{}
	}
	return userData
}

func (fcm *FcmHandler) SubscribeToUser(token, friendId string) {
	if token == "" {
		return
	}
	response, err := fcm.Messaging.SubscribeToTopic(context.Background(), []string{token}, friendId)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(response.SuccessCount, "tokens were subscribed successfully")
	log.Printf("%v tokens were not subscribed", response.FailureCount)
}

func (fcm *FcmHandler) UnsubscribeFromUser(token, topic string) {
	if token == "" {
		return
	}
	response, err := fcm.Messaging.UnsubscribeFromTopic(context.Background(), []string{token}, topic)
	if err != nil {
		log.Printf("Failed to unsubscribe from topic: %v", err)
	}

	fmt.Println(response.SuccessCount, "tokens were unsubscribed successfully")
	log.Printf("%v tokens were not subscribed", response.FailureCount)
}

func (fcm *FcmHandler) SendNotification(token, content string) {
	if token == "" {
		return
	}
	result, err := fcm.Messaging.Send(context.Background(), &messaging.Message{
		Token: token,
		Data: map[string]string{
			"title": content,
		},
	})
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		return
	}
	log.Printf("Successfully sent notification: %v", result)
}

func (fcm *FcmHandler) sendNotificationToFriends(rating RatingEvent) {
	user := fcm.getUserInfo(rating.UserID)
	if user.Friends == nil {
		return
	}
	movieInfo, err := fcm.MongoHandler.FetchFromCache(rating.MovieID)
	if err != nil {
		log.Printf("Failed to fetch movie: %v", err)
	}
	var title string
	if movieInfo.Title != "" {
		title = movieInfo.Title
	} else {
		title = rating.MovieID
	}

	data, _ := json.Marshal(MessageData{Link: fmt.Sprintf("/profile/inspect/%s?from=/", rating.UserID)})
	var content string
	if rating.Multiple {
		content = fmt.Sprintf("%s rated %s and more. See what they thought!.", user.Name, title)
	} else {
		content = fmt.Sprintf("%s rated %s. See what they thought!", user.Name, title)
	}
	result, err := fcm.Messaging.Send(context.Background(), &messaging.Message{
		Topic: rating.UserID,
		Data: map[string]string{
			"title":   fmt.Sprintf("%s rated something.", user.Name),
			"message": content,
			"body":    string(data),
		},
	})
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		return
	}
	log.Printf("Successfully sent notification: %v", result)
}

func (fcm *FcmHandler) handleRatingEvent(rating RatingEvent) {
	fcm.mutex.Lock()
	defer fcm.mutex.Unlock()
	if fcm.userRatings == nil {
		fcm.userRatings = make(map[string]RatingEvent)
	}

	// Check if the user has rated a movie within the last 5 minutes.
	lastRating, ok := fcm.userRatings[rating.UserID]
	if ok && time.Since(lastRating.DateTime) < 5*time.Minute {
		fmt.Println("Notification already scheduled.")
		lastRating.Multiple = true
		return
	}

	log.Printf("Sending notification for rating: %v in 5 minutes", rating)
	// Schedule the notification to be sent after 5 minutes.
	fcm.userRatings[rating.UserID] = rating
	time.AfterFunc(5*time.Minute, func() {
		fcm.mutex.Lock()
		defer fcm.mutex.Unlock()

		// Check if the stored rating is still the same as when it was scheduled.
		storedRating, ok := fcm.userRatings[rating.UserID]
		if ok && storedRating == rating {
			// Send push notification to the user's friends' topics.
			fcm.sendNotificationToFriends(rating)

			// Remove the stored rating to avoid sending duplicate notifications.
			delete(fcm.userRatings, rating.UserID)
		}
	})
}

func (fcm *FcmHandler) AddedTokenWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, token := Handlers.AuthorizationWrapper(w, r, fcm.AuthHandler)
	if !authorized {
		return
	}
	notificationToken := r.URL.Query().Get("token")
	if notificationToken == "" {
		http.Error(w, "No token provided", http.StatusBadRequest)
		return
	}
	friends := fcm.getUserInfo(token.UID).Friends
	if friends != nil {
		for _, friendId := range friends {
			log.Printf("Subscribing to %s", friendId)
			fcm.SubscribeToUser(notificationToken, friendId)
		}
	}
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}
}

func (fcm *FcmHandler) RatedMovieWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, token := Handlers.AuthorizationWrapper(w, r, fcm.AuthHandler)
	if !authorized {
		return
	}

	movieId := r.URL.Query().Get("movieId")
	if movieId == "" {
		http.Error(w, "No movieId provided", http.StatusBadRequest)
		return
	}

	go fcm.handleRatingEvent(RatingEvent{
		UserID:   token.UID,
		MovieID:  movieId,
		DateTime: time.Now(),
	})

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}
}
