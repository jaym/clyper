package processor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/asticode/go-astisub"
	"github.com/jaym/clyper/metadata"
	"github.com/jaym/clyper/objstore"
	"github.com/rs/zerolog/log"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type PreprocessorConfig struct {
	Downscaler        *DownscalerConfig        `mapstructure:"downscaler"`
	Thumbnailer       *ThumbnailerConfig       `mapstructure:"thumbnailr"`
	SubtitleExtractor *SubtitleExtractorConfig `mapstructure:"subtitle_extractor"`
}

type Preprocessor struct {
	config *PreprocessorConfig
}

func NewPreprocessor(cfg PreprocessorConfig) *Preprocessor {
	downscalerWidth := -1
	downscalerHeight := -1
	if cfg.Downscaler != nil {
		if cfg.Downscaler.Width > 0 {
			downscalerWidth = cfg.Downscaler.Width
		}
		if cfg.Downscaler.Height > 0 {
			downscalerHeight = cfg.Downscaler.Height
		}
	}

	thumbnailerWidth := -1
	thumbnailerHeight := -1
	thumbnailerFps := DefaultFramesPerSecond
	if cfg.Thumbnailer != nil {
		if cfg.Thumbnailer.Width > 0 {
			thumbnailerWidth = cfg.Thumbnailer.Width
		}
		if cfg.Thumbnailer.Height > 0 {
			thumbnailerHeight = cfg.Thumbnailer.Height
		}
		if cfg.Thumbnailer.FramesPerSecond > 0 {
			thumbnailerFps = cfg.Thumbnailer.FramesPerSecond
		}
	}

	return &Preprocessor{config: &PreprocessorConfig{
		Downscaler: &DownscalerConfig{
			Width:  downscalerWidth,
			Height: downscalerHeight,
		},
		Thumbnailer: &ThumbnailerConfig{
			Width:           thumbnailerWidth,
			Height:          thumbnailerHeight,
			FramesPerSecond: thumbnailerFps,
		},
		SubtitleExtractor: cfg.SubtitleExtractor,
	}}
}

type ffmpegStreamProbe struct {
	Streams []struct {
		Index     int    `json:"index"`
		CodecType string `json:"codec_type"`
		Tags      struct {
			Language string `json:"language"`
		}
	} `json:"streams"`
}

const (
	EpisodeMetadataVersion = 1
)

var EpisodeMetadataFilename = fmt.Sprintf("METADATA.%d.json", EpisodeMetadataVersion)

type PreprocessorError struct {
	Msg           string
	ffprobeOutput string
}

func (e *PreprocessorError) Error() string {
	return e.Msg
}

func (e *PreprocessorError) VerboseError() string {
	return fmt.Sprintf("FFProbe Output:\n%s\n\n%s", e.Msg, e.ffprobeOutput)
}

func (p *Preprocessor) Process(inputDir string, outputDir string) error {
	log.Info().
		Interface("config", p.config).
		Str("inputDir", inputDir).
		Str("outputDir", outputDir).
		Msg("processing files")

	// The inputDir is the directory containing the video files to be processed.
	// It will be recursively scanned for video files. The video file names
	// must have the season and episode number in the format SXXEXX.

	err := os.MkdirAll(path.Join(outputDir, "internal"), 0755)
	if err != nil {
		return fmt.Errorf("error creating internal output directory: %v", err)
	}

	metadataDbPath := path.Join(outputDir, "internal/metadata.db")
	objStoreReader := objstore.NewLocalFSObjectReader(outputDir)
	metadataDbBuilder, err := metadata.NewDatabaseBuilder(metadataDbPath, objStoreReader)
	if err != nil {
		return fmt.Errorf("error creating metadata database builder: %v", err)
	}
	// Walk the input directory
	err = filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error walking input directory: %v", err)
		}

		if d.IsDir() {
			return nil
		}

		season, episode, err := extractSeasonAndEpisode(path)
		if err != nil {
			log.Warn().Str("path", path).Msg("skipping file")
			return nil
		}

		log.Info().Str("path", path).Int("season", season).Int("episode", episode).Msg("processing file")
		// Process the file
		md, err := p.processFile(path, outputDir, season, episode)
		if err != nil {
			return fmt.Errorf("error processing file: %v", err)
		}

		// Insert the metadata into the database
		err = metadataDbBuilder.AddEpisodeMetadata(*md)
		if err != nil {
			return fmt.Errorf("error adding episode metadata to database: %v", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error processing files: %v", err)
	}

	err = metadataDbBuilder.Build()
	if err != nil {
		return fmt.Errorf("error building metadata database: %v", err)
	}

	return nil
}

func (p *Preprocessor) processFile(inputFilePath string, outputDir string, season int, episode int) (*metadata.EpisodeMetadata, error) {
	probeStr, err := ffmpeg_go.Probe(inputFilePath)
	if err != nil {
		return nil, err
	}

	var probe ffmpegStreamProbe
	err = json.Unmarshal([]byte(probeStr), &probe)
	if err != nil {
		return nil, &PreprocessorError{
			Msg:           fmt.Sprintf("error unmarshalling ffprobe output: %v", err),
			ffprobeOutput: probeStr,
		}
	}

	// Find the english subtitle stream
	subtitlesStream := -1
	videoStream := -1
	for _, stream := range probe.Streams {
		if stream.CodecType == "subtitle" && stream.Tags.Language == "eng" {
			subtitlesStream = stream.Index
			break
		}
		if stream.CodecType == "video" {
			videoStream = stream.Index
		}
	}

	if subtitlesStream == -1 {
		return nil, &PreprocessorError{
			Msg:           "could not find english subtitle stream",
			ffprobeOutput: probeStr,
		}
	}

	if videoStream == -1 {
		return nil, &PreprocessorError{
			Msg:           "could not find video stream",
			ffprobeOutput: probeStr,
		}
	}

	epKey := fmt.Sprintf("%02d/%02d", season, episode)
	episodeMetadataKey := fmt.Sprintf("internal/%s/%s", epKey, EpisodeMetadataFilename)
	subtitlesOutputKey := fmt.Sprintf("internal/%s/subtitles.srt", epKey)
	downscaleOutputKey := fmt.Sprintf("internal/%s/downscale_%d_%d.mkv", epKey, p.config.Downscaler.Width, p.config.Downscaler.Height)

	// Extract the season and episode number from the file name
	episodeMetadata := &metadata.EpisodeMetadata{
		Season:       season,
		Episode:      episode,
		VideoFileKey: downscaleOutputKey,
		SubsFileKey:  subtitlesOutputKey,
	}

	// Write the episode metadata to a file
	episodeMetadataPath := path.Join(outputDir, episodeMetadataKey)

	// Check if the episode has already been processed
	if _, err := os.Stat(episodeMetadataPath); err == nil {
		log.Info().Str("path", inputFilePath).Msg("episode already processed")
		episodeMetadataFile, err := os.Open(episodeMetadataPath)
		if err != nil {
			return nil, fmt.Errorf("error opening episode metadata file: %v", err)
		}
		defer episodeMetadataFile.Close()

		episodeMetadataBytes, err := io.ReadAll(episodeMetadataFile)
		if err != nil {
			return nil, fmt.Errorf("error reading episode metadata file: %v", err)
		}

		err = json.Unmarshal(episodeMetadataBytes, episodeMetadata)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling episode metadata: %v", err)
		}

		return episodeMetadata, nil
	}

	// Ensure the output directories exist
	err = os.MkdirAll(fmt.Sprintf("%s/internal/%s", outputDir, epKey), 0755)
	if err != nil {
		return nil, fmt.Errorf("error creating internal output directory: %v", err)
	}

	err = os.MkdirAll(fmt.Sprintf("%s/public/%s", outputDir, epKey), 0755)
	if err != nil {
		return nil, fmt.Errorf("error creating public output directory: %v", err)
	}

	downscaleOutputPath := path.Join(outputDir, downscaleOutputKey)

	input := ffmpeg_go.Input(inputFilePath)
	downscaleFilter := input.Filter("scale", ffmpeg_go.Args{fmt.Sprintf("%d:%d", p.config.Downscaler.Width, p.config.Downscaler.Height)})
	downscaleSplit := downscaleFilter.Split()
	downscaleOutput := downscaleSplit.Get("0").Output(
		downscaleOutputPath,
		ffmpeg_go.KwArgs{
			"an":     "",
			"c:v":    "libx264",
			"crf":    "18",
			"preset": "fast",
		})
	thumbnailsOutput := downscaleSplit.Get("1").Filter(
		"fps",
		ffmpeg_go.Args{
			fmt.Sprintf("%d", p.config.Thumbnailer.FramesPerSecond),
		},
	).Filter(
		"scale",
		ffmpeg_go.Args{
			fmt.Sprintf("%d:%d", p.config.Thumbnailer.Width, p.config.Thumbnailer.Height),
		},
	).Output(
		path.Join(outputDir, fmt.Sprintf("public/%s/_thumb_%%08d.jpg", epKey)),
		ffmpeg_go.KwArgs{
			"q:v": "1",
		},
	)

	subtitlesOutputPath := path.Join(outputDir, subtitlesOutputKey)
	subtitlesOutput := input.Get(fmt.Sprintf("%d", subtitlesStream)).Output(subtitlesOutputPath)

	err = ffmpeg_go.MergeOutputs(downscaleOutput, thumbnailsOutput, subtitlesOutput).OverWriteOutput().ErrorToStdOut().Run()
	if err != nil {
		return nil, &PreprocessorError{
			Msg: fmt.Sprintf("failed to run ffmpeg on %s: %v", inputFilePath, err),
		}
	}

	// Rename the thumbnails to include the timestamp and remove the leading underscore
	thumbDir := fmt.Sprintf("%s/public/%s", outputDir, epKey)
	files, err := os.ReadDir(thumbDir)
	if err != nil {
		return nil, fmt.Errorf("error listing files in output directory: %v", err)
	}

	thumbnails := []metadata.ThumbMetadata{}
	frameTime := 1000 / p.config.Thumbnailer.FramesPerSecond
	for _, file := range files {
		// Check if the file is a thumbnail
		if !file.IsDir() && file.Name()[0] == '_' {
			// Extract the timestamp from the file name
			_, frameNum, found := strings.Cut(file.Name()[1:], "_")
			frameNum, _, _ = strings.Cut(frameNum, ".")
			if !found {
				return nil, fmt.Errorf("error extracting timestamp from file name: %s", file.Name())
			}
			iFrameNum, err := strconv.Atoi(frameNum)
			if err != nil {
				return nil, fmt.Errorf("error converting frame number to integer: %v", err)
			}
			// Rename the files to include the timestamp and remove the leading underscore
			ts := (iFrameNum * 1000) / p.config.Thumbnailer.FramesPerSecond
			key := fmt.Sprintf("public/%s/thumb_%08d.jpg", epKey, ts)
			newName := path.Join(outputDir, key)

			log.Info().Str("old", file.Name()).Str("new", key).Msg("renaming thumbnail")

			err = os.Rename(path.Join(thumbDir, file.Name()), newName)
			if err != nil {
				return nil, fmt.Errorf("error renaming file: %v", err)
			}

			// Add the file names to the thumbnails slice
			thumbnails = append(thumbnails, metadata.ThumbMetadata{
				Key:   key,
				Start: ts,
				End:   ts + frameTime,
			})
		}
	}
	episodeMetadata.Thumbs = thumbnails

	subtitleMetadata, err := subtitleMetadata(subtitlesOutputPath, p.config.Thumbnailer.FramesPerSecond)
	if err != nil {
		return nil, fmt.Errorf("error extracting subtitle metadata: %v", err)
	}

	episodeMetadata.Subtitles = subtitleMetadata

	episodeMetadataFile, err := os.Create(episodeMetadataPath)
	if err != nil {
		return nil, fmt.Errorf("error creating episode metadata file: %v", err)
	}
	defer episodeMetadataFile.Close()

	episodeMetadataBytes, err := json.Marshal(episodeMetadata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling episode metadata: %v", err)
	}

	_, err = episodeMetadataFile.Write(episodeMetadataBytes)
	if err != nil {
		return nil, fmt.Errorf("error writing episode metadata: %v", err)
	}

	return episodeMetadata, nil
}

var episodeRegex = regexp.MustCompile(`S(\d+)E(\d+)`)

func extractSeasonAndEpisode(inputFilePath string) (int, int, error) {
	matches := episodeRegex.FindStringSubmatch(inputFilePath)
	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("could not extract season and episode number from file name")
	}

	season, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("error converting season number to integer: %v", err)
	}

	episode, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, fmt.Errorf("error converting episode number to integer: %v", err)
	}

	return season, episode, nil
}

func subtitleMetadata(subtitlesPath string, framesPerSecond int) ([]metadata.SubtitleMetadata, error) {
	subtitlesFile, err := os.Open(subtitlesPath)
	if err != nil {
		return nil, fmt.Errorf("error opening subtitles file: %v", err)
	}
	defer subtitlesFile.Close()

	// Parse the subtitles
	subs, err := astisub.ReadFromSRT(subtitlesFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse subtitles")
		return nil, err
	}

	frameTime := int64(1000 / framesPerSecond)
	var subtitleMetadata []metadata.SubtitleMetadata
	for _, itm := range subs.Items {
		startTime := itm.StartAt.Milliseconds()
		endTime := itm.EndAt.Milliseconds()

		// round the startTime down to the nearest frame
		startTime = (startTime / frameTime) * frameTime
		// round the endTime up to the nearest frame
		endTime = ((endTime + frameTime - 1) / frameTime) * frameTime

		var textBuilder strings.Builder
		for _, line := range itm.Lines {
			textBuilder.WriteString(line.String())
			textBuilder.WriteString("\n")
		}
		text := textBuilder.String()

		subtitleMetadata = append(subtitleMetadata, metadata.SubtitleMetadata{
			Start: int(startTime),
			End:   int(endTime),
			Text:  text,
		})
	}

	return subtitleMetadata, nil
}
