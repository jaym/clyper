package clyper

import (
	processor "github.com/jaym/clyper/processors"
	"github.com/spf13/cobra"
)

var makeCmd = &cobra.Command{
	Use: "make",
}

var gifCmd = &cobra.Command{
	Use:  "gif input_file output_file",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		start, _ := cmd.Flags().GetInt("start")
		end, _ := cmd.Flags().GetInt("end")
		text, _ := cmd.Flags().GetString("text")
		fontName, _ := cmd.Flags().GetString("font-name")
		fontColor, _ := cmd.Flags().GetString("font-color")
		fontsDir, _ := cmd.Flags().GetString("fonts-dir")
		desiredMaxSize, _ := cmd.Flags().GetFloat32("desired-max-size")

		err := processor.MakeGif(args[0], args[1], start, end, processor.GifOptions{
			Text:           text,
			FontName:       fontName,
			FontColor:      fontColor,
			FontsDir:       fontsDir,
			DesiredMaxSize: int(desiredMaxSize * 1024 * 1024),
		})
		cobra.CheckErr(err)
	},
}

func init() {
	gifCmd.Flags().Int("start", 0, "start time in milliseconds")
	gifCmd.Flags().Int("end", 0, "end time in milliseconds")
	gifCmd.Flags().String("text", "", "text to overlay on the gif")
	gifCmd.Flags().String("font-name", "", "font name")
	gifCmd.Flags().String("font-color", "", "font color")
	gifCmd.Flags().String("fonts-dir", "", "directory containing fonts")
	gifCmd.Flags().Float32("desired-max-size", 2.0, "desired max size in MB")
	gifCmd.MarkFlagRequired("start")
	gifCmd.MarkFlagRequired("end")

	makeCmd.AddCommand(gifCmd)

	rootCmd.AddCommand(makeCmd)
}
