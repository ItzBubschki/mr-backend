package FirebaseHandlers

import "time"

type Rating struct {
	UserId    string    `firestore:"userId"`
	MovieId   string    `firestore:"movieId"`
	Rating    float64   `firestore:"rating"`
	Comment   string    `firestore:"comment"`
	Timestamp time.Time `firestore:"timestamp"`
	ExpiresAt time.Time `firestore:"expiresAt,omitempty"`
}

type User struct {
	Email            string    `firestore:"email"`
	Friends          []string  `firestore:"friends,omitempty"`
	Name             string    `firestore:"name"`
	Picture          string    `firestore:"picture"`
	RatedMovies      []string  `firestore:"ratedMovies,omitempty"`
	FriendRequests   []string  `firestore:"friendRequests,omitempty"`
	OutgoingRequests []string  `firestore:"outgoingRequests,omitempty"`
	ExpiresAt        time.Time `firestore:"expiresAt,omitempty"`
}
