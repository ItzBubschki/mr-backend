package Handlers

type MovieResponse struct {
	Type          string
	Title         string
	Overview      string
	StreamingInfo *struct {
		De map[string][]platformAvailability
	} `json:"streamingInfo,omitempty"`
	Year          int
	IMDBRating    float64 `json:"imdbRating"`
	IMDBID        string  `json:"imdbId"`
	TMDBRating    float64 `json:"tmdbRating"`
	OriginalTitle string
	BackdropURLs  struct {
		Original string
	}
	Genres []struct {
		Name string
	}
	OriginalLanguage          string
	Runtime                   int
	YoutubeTrailerVideoLink   string `json:"youtubeTrailerVideoLink,omitempty"`
	YoutubeTrailerVideoID     string `json:"youtubeTrailerVideoId,omitempty"`
	PosterURLs                map[string]string
	Tagline                   string
	AdvisedMinimumAudienceAge int
}

type platformAvailability struct {
	Type    string
	Quality string
	Link    string
	Price   *struct {
		Formatted string
		Amount    string
	} `json:"price,omitempty"`
}
