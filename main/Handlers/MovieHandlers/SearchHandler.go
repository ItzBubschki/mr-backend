package MovieHandlers

import (
	"encoding/json"
	"github.com/ItzBubschki/mr-backend/main/Handlers"
	"io"
	"log"
	"net/http"
)

type externalSearchMovieResponse struct {
	Result []MovieResponse `json:"result"`
}

type SearchHandler struct {
	Mongo *MongoHandler
}

func (s *SearchHandler) searchForTerm(search string) []MovieResponse {
	url := Handlers.SearchUrl + search
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-key", Handlers.ApiKey)
	req.Header.Add("x-rapidapi-host", Handlers.ApiHost)

	res, _ := http.DefaultClient.Do(req)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)
	body, _ := io.ReadAll(res.Body)

	var movieResponse externalSearchMovieResponse
	log.Println("Got movies from api")
	err := json.Unmarshal(body, &movieResponse)
	if err != nil {
		return []MovieResponse{}
	}

	go s.Mongo.SaveInCache(movieResponse.Result)

	return movieResponse.Result
}

func (s *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	movies := s.searchForTerm(search)

	// Convert response to JSON
	jsonResponse, err := json.Marshal(movies)
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
