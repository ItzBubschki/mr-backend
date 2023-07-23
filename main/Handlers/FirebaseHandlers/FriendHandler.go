package FirebaseHandlers

import (
	"cloud.google.com/go/firestore"
	"context"
	"firebase.google.com/go/v4/auth"
	"github.com/ItzBubschki/mr-backend/main/Handlers"
	"log"
	"net/http"
)

type FriendHandler struct {
	AuthHandler *auth.Client
	FireStore   *firestore.Client
}

type parseResponse struct {
	user, friend       User
	code               int
	message            string
	userRef, friendRef *firestore.DocumentRef
}

func (f *FriendHandler) authorizationWrapper(w http.ResponseWriter, r *http.Request) (bool, string, string) {
	if r.Method == http.MethodOptions {
		_, _ = w.Write([]byte("OK"))
		return false, "", ""
	} else if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false, "", ""
	}
	idToken := r.Header.Get("Authorization")
	if idToken == "" {
		log.Println("No token found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false, "", ""
	}
	token, err := f.AuthHandler.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		log.Printf("error verifying ID token: %v\n", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false, "", ""
	}

	friendId := r.URL.Query().Get("friendId")
	if friendId == "" {
		http.Error(w, "Missing friendId", http.StatusBadRequest)
		return false, "", ""
	}
	return true, token.UID, friendId
}

func (f *FriendHandler) getAndParse(userId, friendId string) parseResponse {
	user, err := f.FireStore.Collection("Users").Doc(userId).Get(context.Background())
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return parseResponse{code: 404, message: "user doesn't exist"}
	}
	friend, err := f.FireStore.Collection("Users").Doc(friendId).Get(context.Background())
	if err != nil {
		log.Printf("Failed to get friend: %v", err)
		return parseResponse{code: 404, message: "friend doesn't exist"}
	}

	var userData, friendData User
	err = user.DataTo(&userData)
	if err != nil {
		log.Printf("Failed to convert user data: %v", err)
		return parseResponse{code: 500, message: "Internal Server Error"}
	}
	err = friend.DataTo(&friendData)
	if err != nil {
		log.Printf("Failed to convert friend data: %v", err)
		return parseResponse{code: 500, message: "Internal Server Error"}
	}
	return parseResponse{userData, friendData, 200, "Ok", user.Ref, friend.Ref}
}

func (f *FriendHandler) sendFriendRequest(userId, friendId string) (int, string) {
	parsed := f.getAndParse(userId, friendId)
	if parsed.code != 200 {
		return parsed.code, parsed.message
	}
	if Handlers.ArrayContains(parsed.user.Friends, friendId) || Handlers.ArrayContains(parsed.friend.FriendRequests, friendId) {
		return 400, "friend already added"
	}
	if Handlers.ArrayContains(parsed.user.OutgoingRequests, friendId) {
		return 400, "friend request already sent"
	}
	if Handlers.ArrayContains(parsed.user.FriendRequests, friendId) {
		code, message := f.acceptFriendRequest(userId, friendId) //user already has a friend request from that person, so we just accept it
		if code != 200 {
			return code, message
		}
		return 210, "Ok"
	}
	_, err := parsed.friendRef.Update(context.Background(), []firestore.Update{{Path: "friendRequests", Value: firestore.ArrayUnion(userId)}})
	if err != nil {
		log.Printf("Failed to update friend: %v", err)
		return 500, "Internal Server Error"
	}
	_, err = parsed.userRef.Update(context.Background(), []firestore.Update{{Path: "outgoingRequests", Value: firestore.ArrayUnion(friendId)}})
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return 500, "Internal Server Error"
	}
	return 200, "Ok"
}

func (f *FriendHandler) acceptFriendRequest(userId, friendId string) (int, string) {
	parsed := f.getAndParse(userId, friendId)
	if parsed.code != 200 {
		return parsed.code, parsed.message
	}
	if !Handlers.ArrayContains(parsed.user.FriendRequests, friendId) {
		_, err := parsed.userRef.Update(context.Background(), []firestore.Update{{Path: "friendRequests", Value: firestore.ArrayRemove(friendId)}})
		if err != nil {
			log.Printf("Failed to update user: %v", err)
			return 500, "Internal Server Error"
		}
		return 400, "no friend request"
	}

	_, err := parsed.userRef.Update(context.Background(), []firestore.Update{
		{Path: "friends", Value: firestore.ArrayUnion(friendId)},
		{Path: "friendRequests", Value: firestore.ArrayRemove(friendId)},
	})
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return 500, "Internal Server Error"
	}
	_, err = parsed.friendRef.Update(context.Background(), []firestore.Update{
		{Path: "friends", Value: firestore.ArrayUnion(userId)},
		{Path: "outgoingRequests", Value: firestore.ArrayRemove(userId)},
	})
	if err != nil {
		log.Printf("Failed to update friend: %v", err)
		return 500, "Internal Server Error"
	}
	return 200, "Ok"
}

func (f *FriendHandler) declineFriendRequest(userId, friendId string) (int, string) {
	parsed := f.getAndParse(userId, friendId)
	if parsed.code != 200 {
		return parsed.code, parsed.message
	}
	if !Handlers.ArrayContains(parsed.user.FriendRequests, friendId) || !Handlers.ArrayContains(parsed.friend.OutgoingRequests, userId) {
		return 400, "no friend request"
	}
	_, err := parsed.userRef.Update(context.Background(), []firestore.Update{{Path: "friendRequests", Value: firestore.ArrayRemove(friendId)}})
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return 500, "Internal Server Error"
	}
	_, err = parsed.friendRef.Update(context.Background(), []firestore.Update{{Path: "outgoingRequests", Value: firestore.ArrayRemove(userId)}})
	if err != nil {
		log.Printf("Failed to update friend: %v", err)
		return 500, "Internal Server Error"
	}
	return 200, "Ok"
}

func (f *FriendHandler) removeFriend(userId, friendId string) (int, string) {
	parsed := f.getAndParse(userId, friendId)
	if parsed.code != 200 {
		return parsed.code, parsed.message
	}
	if !Handlers.ArrayContains(parsed.user.Friends, friendId) {
		return 400, "Not friends with user"
	}
	_, err := parsed.userRef.Update(context.Background(), []firestore.Update{{Path: "friends", Value: firestore.ArrayRemove(friendId)}})
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return 500, "Internal Server Error"
	}
	_, err = parsed.friendRef.Update(context.Background(), []firestore.Update{{Path: "friends", Value: firestore.ArrayRemove(userId)}})
	if err != nil {
		log.Printf("Failed to update friend: %v", err)
		return 500, "Internal Server Error"
	}
	return 200, "Ok"
}

func (f *FriendHandler) revokeFriendRequest(userId, friendId string) (int, string) {
	parsed := f.getAndParse(userId, friendId)
	if parsed.code != 200 {
		return parsed.code, parsed.message
	}
	if !Handlers.ArrayContains(parsed.user.OutgoingRequests, friendId) {
		return 400, "no friend request sent to this user"
	}
	_, err := parsed.friendRef.Update(context.Background(), []firestore.Update{{Path: "friendRequests", Value: firestore.ArrayRemove(userId)}})
	if err != nil {
		log.Printf("Failed to update friend: %v", err)
		return 500, "Internal Server Error"
	}
	_, err = parsed.userRef.Update(context.Background(), []firestore.Update{{Path: "outgoingRequests", Value: firestore.ArrayRemove(friendId)}})
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return 500, "Internal Server Error"
	}
	return 200, "Ok"
}

func (f *FriendHandler) RevokeRequestWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, uid, friendId := f.authorizationWrapper(w, r)
	if !authorized {
		return
	}

	code, message := f.revokeFriendRequest(uid, friendId)
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}

func (f *FriendHandler) AcceptRequestWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, uid, friendId := f.authorizationWrapper(w, r)
	if !authorized {
		return
	}

	code, message := f.acceptFriendRequest(uid, friendId)
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}

func (f *FriendHandler) DeclineRequestWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, uid, friendId := f.authorizationWrapper(w, r)
	if !authorized {
		return
	}

	code, message := f.declineFriendRequest(uid, friendId)
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}

func (f *FriendHandler) RemoveFriendWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, uid, friendId := f.authorizationWrapper(w, r)
	if !authorized {
		return
	}

	code, message := f.removeFriend(uid, friendId)
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}

func (f *FriendHandler) SendRequestWrapper(w http.ResponseWriter, r *http.Request) {
	authorized, uid, friendId := f.authorizationWrapper(w, r)
	if !authorized {
		return
	}

	code, message := f.sendFriendRequest(uid, friendId)
	w.WriteHeader(code)
	_, _ = w.Write([]byte(message))
}
