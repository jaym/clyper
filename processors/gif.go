package processor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

const MaxGifDurationMS = 10000
const DefaultDisiredMaxSize = 2 * 1024 * 1024

type GifOptions struct {
	Text           string `mapstructure:"text"`
	FontName       string `mapstructure:"font_name"`
	FontColor      string `mapstructure:"font_color"`
	FontsDir       string `mapstructure:"fonts_dir"`
	DesiredMaxSize int    `mapstructure:"desired_max_size"`
}

var ErrInvalidTimeRange = errors.New("invalid time range")

func MakeGif(inputFile string, outputFile string, startTime int, endTime int, opts GifOptions) error {
	if endTime < startTime {
		return ErrInvalidTimeRange
	}

	// 10 seconds is the max we will allow
	if endTime-startTime > MaxGifDurationMS {
		return ErrInvalidTimeRange
	}

	input := ffmpeg_go.Input(inputFile, ffmpeg_go.KwArgs{
		"ss": fmt.Sprintf("%dms", startTime),
		"to": fmt.Sprintf("%dms", endTime),
	})

	// Make a temp directory
	tmpDir, err := os.MkdirTemp("", "clyper")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	// Write srt file
	srtFile := path.Join(tmpDir, "subtitles.srt")
	err = os.WriteFile(srtFile, []byte(fmt.Sprintf("1\n00:00:00,000 --> 00:01:00,000\n%s", strings.ToUpper(opts.Text))), 0644)
	if err != nil {
		return fmt.Errorf("failed to write srt file: %v", err)
	}

	kwargs := ffmpeg_go.KwArgs{}
	forceStyle := []string{"FontSize=24", "Alignment=2", "MarginL=10", "MarginR=10", "MarginV=20"}
	if opts.FontName != "" {
		forceStyle = append(forceStyle, fmt.Sprintf("Fontname=%s", opts.FontName))
	}
	if opts.FontColor != "" {
		forceStyle = append(forceStyle, fmt.Sprintf("PrimaryColour=&H%s", opts.FontColor))
	}
	if len(forceStyle) > 0 {
		kwargs["force_style"] = strings.Join(forceStyle, ",")
	}

	if opts.FontsDir != "" {
		kwargs["fontsdir"] = opts.FontsDir
	}

	withSubs := input.Filter("subtitles", ffmpeg_go.Args{srtFile}, kwargs)

	split := withSubs.Split()
	palette := split.Get("0").Filter("palettegen", ffmpeg_go.Args{"max_colors=64"}).Split()

	multiFpsSplit := split.Get("1").Split()

	outputOrigFpsFile := path.Join(tmpDir, "orig_fps.gif")
	outputOrigFps := ffmpeg_go.Filter([]*ffmpeg_go.Stream{
		multiFpsSplit.Get("0"),
		palette.Get("0"),
	}, "paletteuse", ffmpeg_go.Args{}).
		Output(outputOrigFpsFile)

	output12FpsFile := path.Join(tmpDir, "12_fps.gif")
	output12Fps := ffmpeg_go.Filter([]*ffmpeg_go.Stream{
		multiFpsSplit.Get("1").Filter("fps", ffmpeg_go.Args{"fps=12"}),
		palette.Get("1"),
	}, "paletteuse", ffmpeg_go.Args{}).
		Output(output12FpsFile)

	err = ffmpeg_go.MergeOutputs(outputOrigFps, output12Fps).OverWriteOutput().ErrorToStdOut().Run()
	if err != nil {
		return fmt.Errorf("failed to create gif: %v", err)
	}

	// Select the gif closest to 2MB
	origFpsSize, err := os.Stat(outputOrigFpsFile)
	if err != nil {
		return fmt.Errorf("failed to get file size: %v", err)
	}

	wantedMaxSize := DefaultDisiredMaxSize
	if opts.DesiredMaxSize > 0 {
		wantedMaxSize = opts.DesiredMaxSize
	}

	if origFpsSize.Size() < int64(wantedMaxSize) {
		err = renameOrCopy(outputOrigFpsFile, outputFile)
		if err != nil {
			return fmt.Errorf("failed to rename file: %v", err)
		}
	} else {
		err = renameOrCopy(output12FpsFile, outputFile)
		if err != nil {
			return fmt.Errorf("failed to rename file: %v", err)
		}
	}

	return nil
}

func renameOrCopy(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return copyFile(src, dst)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
