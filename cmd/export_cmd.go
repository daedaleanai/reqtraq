package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/reqs"
	"github.com/pkg/errors"
)

var fExportRaw *bool

var exportCmd = &cobra.Command{
	Use:   "export OUT_DIR",
	Args:  cobra.ExactArgs(1),
	Short: "Export the parsed requirements as JSON",
	Long:  `The parsed requirements exported as JSON can be analyzed, or aggregated with others to produce a complete graph.`,
	RunE:  RunAndHandleError(runExport),
}

// exportedReqsGraph is turned into JSON to be consumed by external clients.
// See the struct with the same name in mdconvert.
type exportedReqsGraph struct {
	Reqs []struct {
		ID        string
		ParentIds []string
		Document  struct {
			Path string
		}
	}
}

// newExportedReqsGraph copies data out of the reqs graph to be exported.
// @llr REQ-TRAQ-SWL-78
func newExportedReqsGraph(reqs *reqs.ReqGraph) exportedReqsGraph {
	data := exportedReqsGraph{
		Reqs: nil,
	}
	ids := make([]string, 0, len(reqs.Reqs))
	for id := range reqs.Reqs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		r := reqs.Reqs[id]
		data.Reqs = append(data.Reqs, struct {
			ID        string
			ParentIds []string
			Document  struct{ Path string }
		}{
			ID:        r.ID,
			ParentIds: r.ParentIds,
			Document: struct {
				Path string
			}{
				Path: r.Document.Path,
			},
		})
	}
	return data
}

// exportReqsGraph writes the specified requirements graph as JSON file.
// @llr REQ-TRAQ-SWL-78
func exportReqsGraph(reqs *reqs.ReqGraph, filePath string, raw bool) error {
	fmt.Println("Exporting to:", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	jsonWriter := json.NewEncoder(file)
	jsonWriter.SetIndent("", "  ")
	if raw {
		if err := jsonWriter.Encode(reqs); err != nil {
			return errors.Wrap(err, "raw graph JSON encoding")
		}
	} else {
		data := newExportedReqsGraph(reqs)
		if err := jsonWriter.Encode(data); err != nil {
			return errors.Wrap(err, "processed graph JSON encoding")
		}
	}
	return file.Close()
}

// the run command for export
// @llr REQ-TRAQ-SWL-78
func runExport(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return errors.Wrap(err, "setup configuration")
	}

	rg, err := reqs.BuildGraph(reqtraqConfig)
	if err != nil {
		return errors.Wrap(err, "build graph")
	}

	exportDir := args[0]
	filePath := path.Join(exportDir, string(rg.ReqtraqConfig.TargetRepo)+".json")
	if err := exportReqsGraph(rg, filePath, *fExportRaw); err != nil {
		return errors.Wrap(err, "export requirements graph")
	}

	return nil
}

// Registers the export command
// @llr REQ-TRAQ-SWL-78
func init() {
	fExportRaw = exportCmd.PersistentFlags().Bool("raw", false, "Export the raw ReqGraph so it can be aggregated with others. UNSTABLE API! Future reqtraq versions will fail to read it.")
	rootCmd.AddCommand(exportCmd)
}
