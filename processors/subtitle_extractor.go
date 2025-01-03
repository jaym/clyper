package processor

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type SubtitleExtractor struct {
}

type SubtitleExtractorConfig struct {
}

// NewSubtitleExtractor creates a new SubtitleExtractor instance.
func NewSubtitleExtractor(cfg SubtitleExtractorConfig) *SubtitleExtractor {
	return &SubtitleExtractor{}
}

type SubtitleMetadata struct {
	// Name is the name of the subtitle.
	Name string `json:"name"`
}

// Run runs the subtitle extractor processor.
func (s *SubtitleExtractor) Run(inputFilePath string, outputDir string) (*SubtitleMetadata, error) {
	subtitlesStream, err := findSubtitleStream(inputFilePath)
	if err != nil {
		return nil, fmt.Errorf("error finding subtitle stream: %v", err)
	}

	outputPath := fmt.Sprintf("%s/subtitles.srt", outputDir)
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputFilePath,
		"-map", fmt.Sprintf("0:%s", subtitlesStream),
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running ffmpeg: %v, output: %s", err, string(output))
	}

	return &SubtitleMetadata{
		Name: "subtitles.srt",
	}, nil
}

var subtitleStreamRegex = regexp.MustCompile(`Stream #0:(\d+)\(eng\): Subtitle:`)

func findSubtitleStream(inputFilePath string) (string, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputFilePath,
	)

	output, _ := cmd.CombinedOutput()

	// Find english subtitle stream
	reader := strings.NewReader(string(output))
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := subtitleStreamRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("no subtitle stream found")
}
