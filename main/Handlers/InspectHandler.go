package Handlers

import (
	"encoding/json"
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

func (i *InspectHandler) searchForSingleMovie(movieId string) MovieResponse {
	url := inspectUrl + movieId
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-key", apiKey)
	req.Header.Add("x-rapidapi-host", apiHost)

	res, _ := http.DefaultClient.Do(req)

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
		return MovieResponse{}
	}

	go i.Mongo.SaveInCache([]MovieResponse{movieResponse.Result})

	return movieResponse.Result
}

func (i *InspectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	movieId := r.URL.Query().Get("movieId")
	var movie MovieResponse
	movie, err := i.Mongo.FetchFromCache(movieId)
	if len(movie.Title) == 0 {
		if err != nil {
			log.Println("Failed to fetch movie from cache:", err)
		}
		movie = i.searchForSingleMovie(movieId)
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
