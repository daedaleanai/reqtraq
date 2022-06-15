package main

import (
	"log"
	"os"
	"regexp"

	"github.com/daedaleanai/cobra"
)

var (
	reportPrefix            *string
	reportTitleFilterString *string
	reportIdFilterString    *string
	reportBodyFilterString  *string
	reportAt                *string
	reportSince             *string
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
	reportTitleFilterString = reportCmd.PersistentFlags().String("title_filter", "", "Regular expression to filter by requirement title.")
	reportIdFilterString = reportCmd.PersistentFlags().String("id_filter", "", "Regular expression to filter by requirement id.")
	reportBodyFilterString = reportCmd.PersistentFlags().String("body_filter", "", "Regular expression to filter by requirement body.")
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

	rg, err := buildGraph(*reportAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *ReqGraph
	if *reportSince != "" {
		prg, err = buildGraph(*reportSince, reqtraqConfig)
		if err != nil {
			return err
		}
	}

	diffs := rg.ChangedSince(prg)

	of, err := os.Create(*reportPrefix + "down.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err := rg.ReportDown(of); err != nil {
		return err
	}
	of.Close()

	filter, err := createFilterFromCmdLine()
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*reportPrefix + "down-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := rg.ReportDownFiltered(of, &filter, diffs); err != nil {
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

	rg, err := buildGraph(*reportAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *ReqGraph
	if *reportSince != "" {
		prg, err = buildGraph(*reportSince, reqtraqConfig)
		if err != nil {
			return err
		}
	}

	diffs := rg.ChangedSince(prg)

	of, err := os.Create(*reportPrefix + "issues.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err := rg.ReportIssues(of); err != nil {
		return err
	}
	of.Close()
	filter, err := createFilterFromCmdLine()
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*reportPrefix + "issues-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := rg.ReportIssuesFiltered(of, &filter, diffs); err != nil {
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

	rg, err := buildGraph(*reportAt, reqtraqConfig)
	if err != nil {
		return err
	}

	var prg *ReqGraph
	if *reportSince != "" {
		prg, err = buildGraph(*reportSince, reqtraqConfig)
		if err != nil {
			return err
		}
	}

	diffs := rg.ChangedSince(prg)

	of, err := os.Create(*reportPrefix + "up.html")
	if err != nil {
		return err
	}
	log.Print("Creating ", of.Name(), " (this may take a while)...")
	if err = rg.ReportUp(of); err != nil {
		return err
	}
	of.Close()

	filter, err := createFilterFromCmdLine()
	if err != nil {
		return err
	}
	if !filter.IsEmpty() || diffs != nil {
		of, err := os.Create(*reportPrefix + "up-filtered.html")
		if err != nil {
			return err
		}
		log.Print("Creating ", of.Name(), " (this may take a while)...")
		if err := rg.ReportUpFiltered(of, &filter, diffs); err != nil {
			return err
		}
		of.Close()
	}

	return nil
}

// createFilterFromCmdLine reads the filter regular expressions from the command line arguments and
// compiles them into a filter structure ready to use
// @llr REQ-TRAQ-SWL-35, REQ-TRAQ-SWL-36
func createFilterFromCmdLine() (ReqFilter, error) {
	filter := ReqFilter{} // Filter for report generation
	var err error
	if len(*reportTitleFilterString) > 0 {
		filter.TitleRegexp, err = regexp.Compile(*reportTitleFilterString)
		if err != nil {
			return filter, err
		}
	}
	if len(*reportIdFilterString) > 0 {
		filter.IDRegexp, err = regexp.Compile(*reportIdFilterString)
		if err != nil {
			return filter, err
		}
	}
	if len(*reportBodyFilterString) > 0 {
		filter.BodyRegexp, err = regexp.Compile(*reportBodyFilterString)
		if err != nil {
			return filter, err
		}
	}
	// TODO can't currently filter on attributes from the command line
	return filter, nil
}
