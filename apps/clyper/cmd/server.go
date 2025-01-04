package clyper

import (
	"net/http"
	"path"

	"github.com/jaym/clyper/api"
	"github.com/jaym/clyper/metadata"
	processor "github.com/jaym/clyper/processors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:  "serve",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		addr, _ := cmd.Flags().GetString("addr")
		dbPath, _ := cmd.Flags().GetString("db")
		objstorePath, _ := cmd.Flags().GetString("objstore")
		fontsDir, _ := cmd.Flags().GetString("fonts-dir")
		fontName, _ := cmd.Flags().GetString("font-name")

		db, err := metadata.OpenDatabase(path.Join(objstorePath, dbPath))
		cobra.CheckErr(err)

		httpHandler := api.NewApiHandler(db, objstorePath, processor.GifOptions{
			FontsDir: fontsDir,
			FontName: fontName,
		})

		log.Info().Str("addr", addr).Msg("Listening")
		err = http.ListenAndServe(addr, httpHandler)
		cobra.CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("addr", ":8991", "address to listen on")
	serveCmd.Flags().String("db", "internal/metadata.db", "path to the database")
	serveCmd.Flags().String("objstore", "objstore", "path to the object store")
	serveCmd.Flags().String("fonts-dir", "", "path to the fonts directory")
	serveCmd.Flags().String("font-name", "", "default font name")

}
