package MovieHandlers

import (
	"encoding/json"
	"errors"
	"github.com/ItzBubschki/mr-backend/main/Handlers"
	"io"
	"log"
	"net/http"
	"strings"
)

type externalInspectMovieResponse struct {
	Result MovieResponse `json:"result"`
}

type InspectHandler struct {
	Mongo *MongoHandler
}

func (i *InspectHandler) searchForSingleMovie(movieId string, country string) (MovieResponse, error) {
	url := Handlers.InspectUrl + movieId + "&country=" + country
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

	movie := movieResponse.Result
	movie.Country = country
	go i.Mongo.SaveInCache([]MovieResponse{movie})

	return movieResponse.Result, nil
}

func (i *InspectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	movieId := r.URL.Query().Get("movieId")
	country := r.URL.Query().Get("country")
	if movieId == "" {
		http.Error(w, "Movie ID is required", http.StatusBadRequest)
		return
	}

	if country == "" {
		country = "de"
	}

	if len(country) != 2 {
		http.Error(w, "Country code must be 2 characters long", http.StatusBadRequest)
		return
	}

	log.Printf("Inspecting movie with ID: %s for country %s \n", movieId, country)

	var movie MovieResponse
	movie, err := i.Mongo.FetchFromCache(movieId, country)
	if len(movie.Title) == 0 {
		if err != nil {
			log.Println("Failed to fetch movie from cache:", err)
		}
		movie, err = i.searchForSingleMovie(movieId, country)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	} else {
		log.Println("Found movie in cache")
	}

	if movie.StreamingInfo[country] == nil {
		log.Println("Movie available in cache but not for requested country")
		movie, err = i.searchForSingleMovie(movieId, country)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	}

	//clear all other countries from the StreamingInfo result to reduce response size
	for key := range movie.StreamingInfo {
		if strings.ToUpper(key) != strings.ToUpper(country) {
			delete(movie.StreamingInfo, key)
		}
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
