package processor

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const DefaultFramesPerSecond = 5

type Thumbnailer struct {
	// framesPerSecond is the rate at which frames are extracted from the video
	framesPerSecond int
	// width is the width of the thumbnail
	width int
	// height is the height of the thumbnail
	height int
}

type ThumbnailerConfig struct {
	// FramesPerSecond is the rate at which frames are extracted from the video
	FramesPerSecond int `mapstructure:"fps"`
	// Width is the width of the thumbnail
	Width int `mapstructure:"width"`
	// Height is the height of the thumbnail
	Height int `mapstructure:"height"`
}

// NewThumbnailer creates a new Thumbnailer instance
func NewThumbnailer(cfg ThumbnailerConfig) *Thumbnailer {
	width := -1
	height := -1
	framesPerSecond := DefaultFramesPerSecond
	if cfg.Width > 0 {
		width = cfg.Width
	}
	if cfg.Height > 0 {
		height = cfg.Height
	}
	if cfg.FramesPerSecond > 0 {
		framesPerSecond = cfg.FramesPerSecond
	}
	return &Thumbnailer{
		framesPerSecond: framesPerSecond,
		width:           width,
		height:          height,
	}
}

type ThumbnailMetadata struct {
	// Name is the name of the thumbnail.
	Name string `json:"name"`
	// Timestamp is the timestamp of the thumbnail.
	Timestamp int `json:"timestamp"`
}

type ThumbnailsMetadata struct {
	Thumbnails []ThumbnailMetadata `json:"thumbnails"`
}

// Run runs the thumbnailer processor.
func (t *Thumbnailer) Run(videoPath string, outputDir string) (*ThumbnailsMetadata, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:%d", t.framesPerSecond, t.width, t.height),
		"-q:v", "1",
		fmt.Sprintf("%s/_thumb_%%08d.jpg", outputDir),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error extracting frames: %v, output: %s", err, string(output))
	}

	thumbnails := []ThumbnailMetadata{}

	// List the files in the output directory
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, fmt.Errorf("error listing files in output directory: %v", err)
	}

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
			ts := (iFrameNum / t.framesPerSecond) * 1000
			newName := fmt.Sprintf("%s/thumb_%08d.jpg", outputDir, ts)
			err = os.Rename(fmt.Sprintf("%s/%s", outputDir, file.Name()), newName)
			if err != nil {
				return nil, fmt.Errorf("error renaming file: %v", err)
			}
			// Add the file names to the thumbnails slice
			thumbnails = append(thumbnails, ThumbnailMetadata{
				Name:      newName,
				Timestamp: ts,
			})
		}
	}

	return &ThumbnailsMetadata{
		Thumbnails: thumbnails,
	}, nil
}
