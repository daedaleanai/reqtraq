package cmd

import (
	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/web"
	"github.com/pkg/errors"
)

var webAddr *string

var webCmd = &cobra.Command{
	Use:   "web [graph.json ...]",
	Short: "Starts a local web server to facilitate interaction with reqtraq",
	Long:  "Starts a local web server to facilitate interaction with reqtraq",
	RunE:  RunAndHandleError(runWebCmd),
}

// Starts the web server listening on the supplied address:port
// @llr REQ-TRAQ-SWL-58
func runWebCmd(command *cobra.Command, args []string) error {
	rg, err := loadReqGraph(args)
	if err != nil {
		return errors.Wrap(err, "load req graph")
	}
	return web.Serve(reqtraqConfig, rg, *webAddr)
}

// Registers the web command
// @llr REQ-TRAQ-SWL-58
func init() {
	webAddr = webCmd.PersistentFlags().String("addr", ":8080", "The ip:port where to serve.")
	rootCmd.AddCommand(webCmd)
}
