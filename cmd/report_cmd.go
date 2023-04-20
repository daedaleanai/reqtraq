package cmd

import (
	"log"
	"os"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/diff"
	"github.com/daedaleanai/reqtraq/report"
	"github.com/daedaleanai/reqtraq/reqs"
)

var (
	reportPrefix          *string
	reportIdFilter        *string
	reportTitleFilter     *string
	reportBodyFilter      *string
	reportAttributeFilter *[]string
	reportAt              *string
	reportSince           *string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Creates an HTML traceability report",
	Long:  "Creates an HTML traceability report",
}

var reportDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Creates an HTML traceability report from system requirements down to code",
	Long:  "Creates an HTML traceability report from system requirements down to code",
	RunE:  runReportDownCmd,
}
var reportUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements",
	Long:  "Creates an HTML traceability report from code, to LLRs, to HLRs and to system requirements",
	RunE:  runReportUpCmd,
}

var reportIssuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Creates an HTML report with all issues found in the requirement documents",
	Long:  "Creates an HTML report with all issues found in the requirement documents",
	RunE:  runReportIssuesCmd,
}

// Registers the report commands
// @llr REQ-TRAQ-SWL-35
func init() {
	reportPrefix = reportCmd.PersistentFlags().String("pfx", "./req-", "Path and filename prefix for reports.")
	reportIdFilter = reportCmd.PersistentFlags().String("id", "", "Regular expression to filter by requirement id.")
	reportTitleFilter = reportCmd.PersistentFlags().String("title", "", "Regular expression to filter by requirement title.")
	reportBodyFilter = reportCmd.PersistentFlags().String("body", "", "Regular expression to filter by requirement body.")
	reportAttributeFilter = reportCmd.PersistentFlags().StringSlice("attribute", nil, "Regular expression to filter by requirement attribute.")
	reportAt = reportCmd.PersistentFlags().String("at", "", "The commit representing the end of the range.")
	reportSince = reportCmd.PersistentFlags().String("since", "", "The commit representing the start of the range.")

	reportCmd.AddCommand(reportUpCmd)
	reportCmd.AddCommand(reportDownCmd)
	reportCmd.AddCommand(reportIssuesCmd)
	rootCmd.AddCommand(reportCmd)
}

// runReportDown creates a requirements graph (and if necessary for comparison a previous graph) and
// generates a top-down html report, showing the implementation for each top-level requirement
// @llr REQ-TRAQ-SWL-35
func runReportDownCmd(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return err
	}

	rg, err := reqs.BuildGraph(*reportAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *reqs.ReqGraph
	if *reportSince != "" {
		prg, err = reqs.BuildGraph(*reportSince, reqtraqConfig)
		if err != nil {
			return err
		}
	}

	diffs := diff.ChangedSince(rg, prg)

	of, err := os.Create(*reportPrefix + "down.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err := report.ReportDown(rg, of); err != nil {
		return err
	}
	of.Close()

	filter, err := reqs.CreateFilter(*reportIdFilter, *reportTitleFilter, *reportBodyFilter, *reportAttributeFilter)
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*reportPrefix + "down-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := report.ReportDownFiltered(rg, of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil

}

// runReportIssues creates a requirements graph (and if necessary for comparison a previous graph) and
// generates an issues html report, showing any validation problems
// @llr REQ-TRAQ-SWL-36
func runReportIssuesCmd(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return err
	}

	rg, err := reqs.BuildGraph(*reportAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *reqs.ReqGraph
	if *reportSince != "" {
		prg, err = reqs.BuildGraph(*reportSince, reqtraqConfig)
		if err != nil {
			return err
		}
	}

	diffs := diff.ChangedSince(rg, prg)

	of, err := os.Create(*reportPrefix + "issues.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err := report.ReportIssues(rg, of); err != nil {
		return err
	}
	of.Close()
	filter, err := reqs.CreateFilter(*reportIdFilter, *reportTitleFilter, *reportBodyFilter, *reportAttributeFilter)
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*reportPrefix + "issues-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := report.ReportIssuesFiltered(rg, of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil
}

// runReportUp creates a requirements graph (and if necessary for comparison a previous graph) and
// generates a bottom-up html report, showing the top-level requirement for each implemented function
// @llr REQ-TRAQ-SWL-35
func runReportUpCmd(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return err
	}

	rg, err := reqs.BuildGraph(*reportAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *reqs.ReqGraph
	if *reportSince != "" {
		prg, err = reqs.BuildGraph(*reportSince, reqtraqConfig)
		if err != nil {
			return err
		}
	}

	diffs := diff.ChangedSince(rg, prg)

	of, err := os.Create(*reportPrefix + "up.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err = report.ReportUp(rg, of); err != nil {
		return err
	}
	of.Close()

	filter, err := reqs.CreateFilter(*reportIdFilter, *reportTitleFilter, *reportBodyFilter, *reportAttributeFilter)
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*reportPrefix + "up-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := report.ReportUpFiltered(rg, of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil
}