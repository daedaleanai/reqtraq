// Daedalean-specific configuration file, defining a bunch of constants that are company-specific. Create your own and
// mark it with your build tag, then remove the !ddln tag below.
// +build ddln !ddln

package config

// Project name
const ProjectName = "Reqtraq"

type RequirementLevel int

// Requirement levels according to DO-178C (do not change!)
const (
	SYSTEM RequirementLevel = iota
	HIGH
	LOW
	CODE
)

// Document types:
// ORD - Overall (aka System) Requirement Document
// SRD - Software Requirements Data
// SDD - Software Design Description
// HRD - Hardware Requirements Data
// HDD - Hardware Design Description

// Requirement types:
// SYS - System/overall requirements (defined in ORD documents)
// SWH - Sofware high-level requirements (defined in SRD documents)
// SWL - Software low-level requirements (defined in SDD documents)
// HWH - Hardware high-level requirements (defined in HRD documents)
// HWL - Hardware low-level requirements (defined in HDD documents)

// Map from requirement type to requirement level.
var ReqTypeToReqLevel = map[string]RequirementLevel{
	"SYS": SYSTEM,
	"SWH": HIGH,
	"HWH": HIGH,
	"SWL": LOW,
	"HWL": LOW,
}

// Map from document type to requirement type.
var DocTypeToReqType = map[string]string{
	"ORD": "SYS",
	"SRD": "SWH",
	"HRD": "HWH",
	"SDD": "SWL",
	"HDD": "HWL",
}

// Map from requirement type to document ID and document type.
var ReqTypeToDocIdAndType = map[string]string{
	"SYS": "100-ORD",
	"SWH": "211-SRD",
	"SWL": "212-SDD",
	"HWH": "311-HRD",
	"HWL": "312-HDD",
}

// Map from document type to document ID.
// TODO: clean up numbers, remove duplicates.
var DocTypeToDocId = map[string]string{
	"H":      "0",
	"DS":     "1",
	"SRS":    "6",
	"SDS":    "7",
	"SCS":    "8",
	"HRS":    "9",
	"HCS":    "10",
	"DAS":    "11",
	"HDS":    "12",
	"HVVS":   "13",
	"HAS":    "14",
	"HCMS":   "15",
	"TAS":    "34",
	"ORD":    "100",
	"SP":     "150",
	"SFA":    "151",
	"PSAC":   "200",
	"SCMP":   "201",
	"SQAP":   "202",
	"SDP":    "203",
	"SVP":    "204",
	"TQP":    "205",
	"SAS":    "206",
	"SRD":    "211",
	"SDD":    "212",
	"SVCP":   "213",
	"PHAK":   "300",
	"HRD":    "311",
	"HDD":    "312",
	"CLPSAC": "101",
	"CLSDP":  "102",
	"CLSVP":  "103",
	"CLSCMP": "104",
	"CLSQAP": "105",
	"CLSDD":  "107",
	"CLSRD":  "106",
	"CLSVCP": "108",
	"CLSCI":  "109",
	"CLTQP":  "110",
	"CLSAS":  "111",
	"TPPSAC": "201",
	"TPSRD":  "206",
	"TPSDD":  "207",
	"TPSVCP": "208",
	"TPHRD":  "209",
	"TPORD":  "210",
	"TPSFHA": "211",
	"TPFFPA": "212",
}
