package metadata

type EpisodeMetadata struct {
	// Season is the season number of the episode.
	Season int `json:"season"`
	// Episode is the episode number of the episode.
	Episode      int                `json:"episode"`
	Thumbs       []ThumbMetadata    `json:"thumbs"`
	Subtitles    []SubtitleMetadata `json:"subtitles"`
	VideoFileKey string             `json:"video_file_key"`
	SubsFileKey  string             `json:"subs_file_key"`
}

type ThumbMetadata struct {
	Key   string `json:"key"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type SubtitleMetadata struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}
