package processor

import (
	"fmt"
	"os/exec"
)

type Downscaler struct {
	// width is the width of the thumbnail
	width int
	// height is the height of the thumbnail
	height int
}

type DownscalerConfig struct {
	// Width is the width of the thumbnail
	Width int `mapstructure:"width"`
	// Height is the height of the thumbnail
	Height int `mapstructure:"height"`
}

// NewDownscaler creates a new Downscaler instance
func NewDownscaler(cfg DownscalerConfig) *Downscaler {
	width := -1
	height := -1
	if cfg.Width > 0 {
		width = cfg.Width
	}
	if cfg.Height > 0 {
		height = cfg.Height
	}
	return &Downscaler{
		width:  width,
		height: height,
	}
}

type DownscalerMetadata struct {
	// Name is the name of the thumbnail.
	Name string `json:"name"`
}

// Run runs the downscaler processor.
func (d *Downscaler) Run(inputFilePath string, outputDir string) (*DownscalerMetadata, error) {
	fname := "downscaled.mp4"
	outputPath := fmt.Sprintf("%s/%s", outputDir, fname)
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputFilePath,
		"-vf", fmt.Sprintf("scale=%d:%d", d.width, d.height),
		"-an",
		"-c:v", "libx264",
		"-crf", "18",
		"-preset", "fast",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running ffmpeg: %v, output: %s", err, string(output))
	}
	return &DownscalerMetadata{Name: fname}, nil
}
