package clyper

import (
	"encoding/json"

	processor "github.com/jaym/clyper/processors"
	"github.com/spf13/cobra"
)

var preprocessCmd = &cobra.Command{
	Use: "preprocess",
}

var subs = &cobra.Command{
	Use:  "subs input_file output_dir",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		p := processor.NewSubtitleExtractor(processor.SubtitleExtractorConfig{})
		meta, err := p.Run(args[0], args[1])
		cobra.CheckErr(err)
		// Pretty print the metadata as json
		o, _ := json.MarshalIndent(meta, "", "  ")
		cmd.Println(string(o))
	},
}

var thumbs = &cobra.Command{
	Use:  "thumbs [OPTIONS] input_file output_dir",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		width, _ := cmd.Flags().GetInt("width")
		height, _ := cmd.Flags().GetInt("height")

		p := processor.NewThumbnailer(processor.ThumbnailerConfig{
			Width:  width,
			Height: height,
		})

		meta, err := p.Run(args[0], args[1])
		cobra.CheckErr(err)
		// Pretty print the metadata as json
		o, _ := json.MarshalIndent(meta, "", "  ")
		cmd.Println(string(o))
	},
}

var downscale = &cobra.Command{
	Use:  "downscale [OPTIONS] input_file output_dir",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		width, _ := cmd.Flags().GetInt("width")
		height, _ := cmd.Flags().GetInt("height")

		p := processor.NewDownscaler(processor.DownscalerConfig{
			Width:  width,
			Height: height,
		})

		meta, err := p.Run(args[0], args[1])
		cobra.CheckErr(err)
		// Pretty print the metadata as json
		o, _ := json.MarshalIndent(meta, "", "  ")
		cmd.Println(string(o))
	},
}

var run = &cobra.Command{
	Use:  "run input_dir output_dir",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		p := processor.NewPreprocessor(processor.PreprocessorConfig{
			Downscaler: &processor.DownscalerConfig{
				Width: 640,
			},
			Thumbnailer: &processor.ThumbnailerConfig{
				Width: 320,
			},
			SubtitleExtractor: &processor.SubtitleExtractorConfig{},
		})

		err := p.Process(args[0], args[1])
		cobra.CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(preprocessCmd)
	preprocessCmd.AddCommand(subs)

	thumbs.Flags().Int("width", 320, "Width of the thumbnail")
	thumbs.Flags().Int("height", -1, "Height of the thumbnail")
	thumbs.Flags().Int("fps", 1, "Frames per second")
	preprocessCmd.AddCommand(thumbs)

	downscale.Flags().Int("width", 640, "Width of the thumbnail")
	downscale.Flags().Int("height", -1, "Height of the thumbnail")
	preprocessCmd.AddCommand(downscale)

	preprocessCmd.AddCommand(run)
}
