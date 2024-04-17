package MovieHandlers

import (
	"encoding/json"
	"errors"
	"github.com/ItzBubschki/mr-backend/main/Handlers"
	"io"
	"log"
	"net/http"
)

type externalInspectMovieResponse struct {
	Result MovieResponse `json:"result"`
}

type InspectHandler struct {
	Mongo *MongoHandler
}

func (i *InspectHandler) searchForSingleMovie(movieId string) (MovieResponse, error) {
	url := Handlers.InspectUrl + movieId
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-key", Handlers.ApiKey)
	req.Header.Add("x-rapidapi-host", Handlers.ApiHost)

	res, _ := http.DefaultClient.Do(req)
	
	if res.StatusCode != 200 {
		return MovieResponse{}, errors.New("received non-200 status code")
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)
	body, _ := io.ReadAll(res.Body)

	var movieResponse externalInspectMovieResponse
	log.Println("Got movie from api")
	err := json.Unmarshal(body, &movieResponse)
	if err != nil {
		return MovieResponse{}, err
	}

	go i.Mongo.SaveInCache([]MovieResponse{movieResponse.Result})

	return movieResponse.Result, nil
}

func (i *InspectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	movieId := r.URL.Query().Get("movieId")
	if movieId == "" {
		http.Error(w, "Movie ID is required", http.StatusBadRequest)
		return
	}

	var movie MovieResponse
	movie, err := i.Mongo.FetchFromCache(movieId)
	if len(movie.Title) == 0 {
		if err != nil {
			log.Println("Failed to fetch movie from cache:", err)
		}
		movie, err = i.searchForSingleMovie(movieId)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	} else {
		log.Println("Found movie in cache")
	}

	// Convert response to JSON
	jsonResponse, err := json.Marshal(movie)
	if err != nil {
		log.Println("Failed to marshal JSON response:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set response headers and write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonResponse)
	if err != nil {
		return
	}
}
