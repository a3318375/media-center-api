package media

// Source identifies the upstream provider that owns SourceID.
type Source string

const (
	SourceTMDB    Source = "tmdb"
	SourceDouban  Source = "douban"
	SourceBangumi Source = "bangumi"
	SourceAniList Source = "anilist"
	SourceEmby    Source = "emby"
)

// Type is our normalized media category, independent of provider terminology.
type Type string

const (
	TypeMovie       Type = "movie"
	TypeTV          Type = "tv"
	TypeAnime       Type = "anime"
	TypeDocumentary Type = "documentary"
	TypePerson      Type = "person"
)

// Identity keeps our stable key separate from an upstream provider identifier.
type Identity struct {
	MediaID   string `json:"mediaId"`
	Source    Source `json:"source"`
	SourceID  string `json:"sourceId"`
	MediaType Type   `json:"mediaType"`
}

type Rating struct {
	Source    Source  `json:"source"`
	Value     float64 `json:"value"`
	VoteCount int64   `json:"voteCount,omitempty"`
}

// Summary is the normalized API model used by discovery cards and search results.
type Summary struct {
	Identity
	Title            string   `json:"title"`
	OriginalTitle    string   `json:"originalTitle,omitempty"`
	Year             *int     `json:"year,omitempty"`
	ReleaseDate      string   `json:"releaseDate,omitempty"`
	Overview         string   `json:"overview,omitempty"`
	Genres           []string `json:"genres"`
	OriginalLanguage string   `json:"originalLanguage,omitempty"`
	OriginCountries  []string `json:"originCountries,omitempty"`
	PosterPath       *string  `json:"posterPath,omitempty"`
	BackdropPath     *string  `json:"backdropPath,omitempty"`
	Ratings          []Rating `json:"ratings"`
}

// Detail extends a summary with fields only needed on a work detail page.
type Detail struct {
	Summary
	RuntimeMinutes *int              `json:"runtimeMinutes,omitempty"`
	Status         string            `json:"status,omitempty"`
	Tagline        string            `json:"tagline,omitempty"`
	ExternalIDs    map[Source]string `json:"externalIds,omitempty"`
	Credits        []Credit          `json:"credits,omitempty"`
	Seasons        []Season          `json:"seasons,omitempty"`
}

type Credit struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	PersonID string `json:"personId,omitempty"`
}

type Season struct {
	SeasonNumber int    `json:"seasonNumber"`
	EpisodeCount *int   `json:"episodeCount,omitempty"`
	Name         string `json:"name,omitempty"`
}
