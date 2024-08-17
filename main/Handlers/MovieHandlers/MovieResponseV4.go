package MovieHandlers

type MovieResponseV4 struct {
	Type          string `json:"showType"`
	Title         string
	Overview      string
	StreamingInfo *struct {
		De []platformAvailabilityV4
	} `json:"streamingOptions,omitempty"`
	Year          int     `json:"releaseYear"`
	IMDBRating    float64 `json:"rating"`
	IMDBID        string  `json:"imdbId"`
	OriginalTitle string
	ImageSet      struct {
		VerticalPoster     map[string]string
		HorizontalPoster   map[string]string
		VerticalBackdrop   map[string]string
		HorizontalBackdrop map[string]string
	}
	Genres []struct {
		Name string
	}
	Runtime int
}

type platformAvailabilityV4 struct {
	Type    string
	Quality string
	Link    string
	Price   *struct {
		Formatted string
		Amount    string
		Currency  string
	} `json:"price,omitempty"`
	Service *struct {
		Name     string
		ImageSet struct {
			LightThemeImage string
			DarkThemeImage  string
		}
	}
}
