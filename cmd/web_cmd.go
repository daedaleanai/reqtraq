package cmd

import (
	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/web"
)

var webAddr *string

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Starts a local web server to facilitate interaction with reqtraq",
	Long:  "Starts a local web server to facilitate interaction with reqtraq",
	RunE:  runWebCmd,
}

// Starts the web server listening on the supplied address:port
// @llr REQ-TRAQ-SWL-58
func runWebCmd(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return err
	}

	return web.Serve(reqtraqConfig, *webAddr)
}

// Registers the web command
// @llr REQ-TRAQ-SWL-58
func init() {
	webAddr = webCmd.PersistentFlags().String("addr", ":8080", "The ip:port where to serve.")
	rootCmd.AddCommand(webCmd)
}
