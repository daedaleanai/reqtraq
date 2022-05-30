# Design Description for Reqtraq

## Introduction

The software architecture and <abbr title="Low-Level Requirement">LLR</abbr>s for Reqtraq are created based on the parent requirements in the <abbr title="High-Level Requirement">HLR</abbr> document.

### Purpose

The purpose of this document is to define how the Reqtraq features should be implemented. This document contains the definition of the software architecture and the software <abbr title="Low-Level Requirement">LLR</abbr>s for the Reqtraq tool that will satisfy the software <abbr title="High-Level Requirement">HLR</abbr>s specified in TRAQ-137-SRD. It follows Section 11.10 of DO-178C / ED-12C.

The purpose of the Reqtraq tool is to comply with the DO-178C / ED-12C traceability data requirements.

### Scope
This document defines the software architecture and low-level requirements for the Reqtraq tool, namely how the tool implementation should meet the high-level requirements.

### Applicable Documents

#### External Documents

**RTCA DO-178C / EUROCAE ED-12C** Software Considerations in Air-borne Systems and Equipment Certification.

#### Internal Documents

**DDLN-1-DS** Documentation Standards

**DDLN-6-RS** Requirements Standards

**TRAQ-100-ORD** Overall Requirements Document for Reqtraq

**TRAQ-137-SRD** Software Requirements Document for Reqtraq

### Nomenclature and Description of Terms

#### Description of Terms

ATX Heading
  : Headings defined within markdown documents are defined ATX-style, meaning text is prefixed with one to six pound signs (hash symbols, #). The level of heading is determined according to how many pound signs were used. See [ATX heading](https://github.github.com/gfm/#atx-headings) for more information.

Markdown
  : The markup language used to format text documents (such as this one). Specifically [GitHub Flavored Markdown](https://github.github.com/gfm) is used.

## Reqtraq Software Overview

### Inputs and Outputs

Data comes in from requirements documents, written in markdown, and source code files in the same Git repository.

Reqtraq has the following outputs:
- Traceability reports and issues, including hyperlinking between documents, either to html documents or served on a web server
- The next available requirement <abbr title="Identification">ID</abbr> for a given document, to the command line
- An abridged version of the requirements from a given document, to the command line

### Software Design and Implementation Details

Documents (markdown and source code) are discovered and parsed from the current Git repository. Each requirement found is parsed into an instance of the `Req` data structure, and each code function in the `Code` data structure. They are all held within a `ReqGraph` object, which is a dictionary of requirements by IDs at a particular Git commit.

Various actions, such as comparison or filtering, are then performed on the `ReqGraph` object, depending on the type of output requested. The requested output is then generated.

ReqGraph source code is arranged as follows:
- main.go: The main entry point to the program, deals with parsing command line arguments and acting accordingly
- req.go: The top-level functions dealing with finding and discovering markdown and source code files
    - parsing.go: Reading and parsing markdown files
    - code.go: Reading and parsing source code files
- diff.go: Comparison of ReqGraph objects
- report.go: Generating html reports to save to disk or provide to a web server
- matrices.go: Generating traceability tables to provide to a web server
- webapp.go: Launch and service a local web server
- git/git.go: Wrapper functions for git commands used within Reqtraq
- linepipes/run.go: Wrapper functions the golang command interface

## Low-level Software Requirements Identification

### code.go

Functions which deal with source code files. Source code is discovered within a given path and searched for functions and associated descriptions. The external program Universal Ctags is used to scan for functions.

#### REQ-TRAQ-SWL-6 Scan for source code

Reqtraq SHALL scan all folders within a given path, relative to the git repository root, searching for supported source code files (those with an extension: c, h, cc, hh or go).

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-7 Skip testdata

Reqtraq SHALL skip any folder named "testdata" when searching for source code.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale: testdata folder may contain source code, which should not be included in the traceability.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-8 Ctags

Reqtraq SHALL use the Universal Ctags application to parse supported source code files (those with an extension: c, h, cc, hh or go) for function names, and store them.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-9 Comments

Reqtraq SHALL scan source code containing functions for comments preceding the functions, and store them.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-38 Source code links

Reqtraq SHALL include links to source code within the reports.

##### Attributes:
- Parents:
- Rationale: Allows quick navigation from requirements to code.
- Verification: Test
- Safety Impact: None

### diff.go

Functions which compare two requirements graphs and return a map-of-slice-of-strings structure which describe how they differ.

Reqtraq generates a list of all requirements changed between checked in versions of the project’s documentation, for use as input to the high-to-low and low-to-high tracing functions. The report generation described in REQ-TRAQ-SWL-12 will be able to receive the following inputs:

- no input: in this case Reqtraq will generate a global HTML report listing all the requirements, from system to high level to low level, defined in the project associated with the repository (each project has its own Git repository)
- a list of requirement IDs (system, high level or low level): in this case the report will be generated for the given requirements, plus all their direct/indirect children. The direct/indirect parent requirements will also be listed, but all children other than the ones requested will be omitted.
- two git commit IDs (or git refs): in this case the report will contain all requirements that were changed between the given commits. If the 2nd commit id is missing, the current state of the repository is used.

Suggested usage (the directory in which Reqtraq is run determines the project for which the report is generated):

```
reqtraq report 
reqtraq report REQ-TRAQ-SWL-OLD-8,REQ-TRAQ-SWL-OLD-9
reqtraq report d6cd1e2bd19e03a81132a23b2025920577f84e37
```
Note: a requirement is considered "changed" if either it was directly edited or one of the child requirements was edited.

#### REQ-TRAQ-SWL-18 Change impact tracing

Reqtraq SHALL compare two requirement graphs by walking them and building a map of changed requirements to descriptions on how they differ.

##### Attributes:
- Parents: REQ-TRAQ-SWH-8
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-40 Ignore non-letter characters

Reqtraq SHALL not report differences to non letter characters when comparing requirements.

##### Attributes:
- Parents:
- Rationale: Squelching non-letter characters provides a meaningful report of differences between requirements.
- Verification: Test
- Safety Impact: None

### git/git.go

Wrapper functions for git commands used within Reqtraq.

#### REQ-TRAQ-SWL-16 Wrap git commands

Reqtraq SHALL wrap git commands needed with go functions.

##### Attributes:
- Parents: REQ-TRAQ-SWH-7
- Rationale:
- Verification: Test
- Safety Impact: None

### linepipes/run.go

Wrapper functions for the golang command interface.

#### REQ-TRAQ-SWL-48 Wrap command interface

Reqtraq SHALL wrap the golang command interface.

##### Attributes:
- Parents:
- Rationale: To provide a simple, consistent way to run external commands and receive output and errors.
- Verification: Test
- Safety Impact: None

### main.go

#### REQ-TRAQ-SWL-32 CLI tool usage

Reqtraq SHALL provide a command line option to show tool usage.

##### Attributes:
- Parents: REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-33 CLI list requirements

Reqtraq SHALL provide a command line option to list the requirements within a given requirements document.

##### Attributes:
- Parents: REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-34 CLI next ID

Reqtraq SHALL provide a command line option to provide the user with the next available ID in a given requirements document.

##### Attributes:
- Parents: REQ-TRAQ-SWH-12, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-35 CLI reports

Reqtraq SHALL provide a command line options to generate reports showing top-down or bottom-up views on the requirements graph.

##### Attributes:
- Parents: REQ-TRAQ-SWH-4, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-36 CLI validate

Reqtraq SHALL provide a command line options to validate a requirements graph and output to the terminal or to a report.

##### Attributes:
- Parents: REQ-TRAQ-SWH-14, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-17 Checkout alternative version

Reqtraq SHALL create a temporary version of an old version of the requirements documents to allow them to be loaded.

##### Attributes:
- Parents: REQ-TRAQ-SWH-8
- Rationale:
- Verification: Test
- Safety Impact: None

### matrices.go

Functions which generate trace matrix tables between different levels of requirements and source code.

#### REQ-TRAQ-SWL-14 Requirement traceability tables

Reqtraq SHALL generate tables which map between two adjacent level of requirements (e.g. system to high-Level software requirements).

##### Attributes:
- Parents: REQ-TRAQ-SWH-5
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-15 Code traceability tables

Reqtraq SHALL generate tables which map between source code functions (source file + function name) and low-level software requirements.

##### Attributes:
- Parents: REQ-TRAQ-SWH-5
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-42 Table sorting by column

Reqtraq SHALL sort tables by the first column and then by the second.

##### Attributes:
- Parents:
- Rationale: It just makes sense to order this way.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-43 Column sorting by requirement number

Where a column contains requirements Reqtraq SHALL sort by requirement number.

##### Attributes:
- Parents:
- Rationale: It just makes sense to order this way.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-44 Column sorting by source code

Where a column contains source code Reqtraq SHALL sort alphabetically by source file and then line number of the function.

##### Attributes:
- Parents:
- Rationale: It just makes sense to order this way.
- Verification: Test
- Safety Impact: None

### parsing.go

Functions for parsing requirements out of markdown documents.

The entry point is ParseCertdoc which in turns calls other functions as follows:
- ParseCertdoc: Checks filename is valid then calls:
- parseMarkdown: Scans file one line at a time looking for requirements that either formatted within ATX headings
                 or held in tables. For each ATX requirement or table calls:
- parseMarkdownFragment: Depending on the type of requirement calls one of the following functions.
- parseReq: Parses ATX heading requirements into the Req structure and returns it.
- parseReqTable: Parses a requirements table and reads each row into a Req structure, returned in a slice.

TODO: include parsing of number/title/body

Reqtraq reads the attributes of each requirement from the attributes section at the end of the requirement definition. The attributes section starts with an [ATX heading](https://github.github.com/gfm/#atx-headings) with the title "Attributes:". Since attributes belong to a requirement, the attribute ATX heading level will be higher than the containing requirement heading level. For example, a requirement defined with an ATX heading level of 3 will have attributes with ATX heading level 4 or higher.

The requirement attributes are formatted one per line:

    ##### Attributes:
    - NAME1: VALUE1
    - NAME2: VALUE2

Attributes can be optional or mandatory. Each attribute has a name. Each attribute may have an associated regular expression to test for validity. Attributes are specified in an `attributes.json` file in the `certdocs` directory. For example, the `attributes.json` for the current document would be:

```
{ "attributes": [
  { "name": "Parent", "optional": false }, 
  { "name": "Verification", "value": "(Demonstration|Unit Test)",
    "optional": false },
  { "name": "Safety Impact", "optional": false } ] }
```

Reqtraq reads the attributes of each requirement held in a requirement table from each column of the table. The first row of the table contains the attribute name for each column, the first column being "ID" to represent requirement ID. The second row is a delimiter. The third row onward contains the requirement ID and associated attribute text as shown:

```
> | ID | Title | Body | Attribute1 | Attribute2 |
> | --- | --- | --- | --- | --- |
> | ReqID1 | <text> | <text> | <text> | <text> |
> | ReqID2 | <text> | <text> | <text> | <text> |
```

As with ATX headings, attributes can be optional or mandatory as specified in the `attributes.json` file.

#### REQ-TRAQ-SWL-24 Certification document names

Reqtraq SHALL check the validity of a certification document name by checking the parts delimited by `-`:

1. Project abbreviation, which shall be the same for all the certification documents of a system, e.g. "TRAQ"
2. Document type sequence number, e.g. "138"
3. Document type, e.g. "SDD"

##### Attributes:
- Parents: REQ-TRAQ-SWH-11
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-2 Requirement definition detection (ATX heading)

Reqtraq SHALL discover requirements definitions in certification documents `.md` files by detecting the beginning and end:
  - a requirement starts with an [ATX heading](https://github.github.com/gfm/#atx-headings) which has at the beginning a requirement ID
  - a requirement ends when another requirement starts, or when a higher-level heading starts.

##### Attributes:
- Parents: REQ-TRAQ-SWH-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-3 Requirement parsing (ATX heading)

Reqtraq SHALL parse requirements text found following an ATX heading into a requirements definition.

##### Attributes:
- Parents: REQ-TRAQ-SWH-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-4 Requirement definition detection (tables)

Reqtraq SHALL discover requirements definitions held in [tables]( https://github.github.com/gfm/#tables-extension-) in certification documents `.md` files by detecting the beginning and end:
  - a requirement table starts with a header row where the first column has the text "ID"
  - a requirement table ends when a non-table row is encountered.

##### Attributes:
- Parents: REQ-TRAQ-SWH-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-5 Requirement parsing (tables)

Reqtraq SHALL parse requirements text found in a table row into a requirements definition.

##### Attributes:
- Parents: REQ-TRAQ-SWH-1
- Rationale:
- Verification: Test
- Safety Impact: None

### report.go

Functions for generating HTML reports showing trace data.

Requirements bodies can include code that is delimited with the triple-backtick delimiter (\`\`\`), which results in rendered HTML as follows:
```
int main(int argc, char** argv) {
        FlyAirplane();
}
```

Requirements bodies can include math (both inline and display) that is delimited with the single dollar sign (\$) for inline, and the double dollar
sign (\$\$) for display. They will be rendered in HTML reports using MathJax and look as follows:

* Inline math: $x=y$
* Display math:
$$
\frac{d}{dx}\left( \int_{0}^{x} f(u)\,du\right)=f(x).
$$

Requirements bodies can include tables in any of the four pandoc table formats. An example simple table is shown here:

```
--------------------------------------------------------------------
             *Alpha*         *Beta*              *Gamma*
----------   -------------   -----------------   --------------------
    Monday     3 Watts        2 pints             3 chickens,
                                                  1 hr at charger

   Tuesday    14 Kilograms    1 Penguin,          1 cheese sandwich
                              2 Thunderbird

 Wednesday    2 aspirin       Tall space ship    (can't remember)
---------------------------------------------------------------------

Table:  *Table of nonsense*, deluxe edition
```

Given a list of requirements, Reqtraq can generate parent and child requirements and code ordered from system to high level to low level requirement to implementation and test, including missing continuations.

**Report structure**

The information will be organized as following:

- SYSTEM\_REQUIREMENT\_1

    - Status: not started/started (% complete)/completed

    - Open issues: &lt;number of open issues&gt;

    - HIGH\_LEVEL\_REQUIREMENT\_1

        - Status: not started/started (% complete)/completed

        - Open issues: &lt;number of open issues&gt;

        - LOW\_LEVEL\_REQUIREMENT\_1

            - Status: not started/started (% complete)/completed

            - Open issues: enumerate open issue IDs

            - Source files: list the source files implementing this requirement

        - LOW\_LEVEL\_REQUIREMENT\_2

            - ...

    - HIGH\_LEVEL\_REQUIREMENT\_2

        - ...

- SYSTEM\_REQUIREMENT\_2

    - ...

**Note 1:** The above list will show "denormalized" requirements, in the sense that if a requirement has multiple parents, it will be listed under each parent. To facilitate readability, the full information will be displayed only the first time a requirement appears, otherwise a link to the first occurrence is used.

**Note 2:** The completion status of SYS, SWH or HWH is:
- not started, if none of the children are started
- started, if at least one child is started
- completed, if \*all\* children are completed

Given a list of requirements, Reqtraq shall generate parent and child requirements and code, ordered from implementation or test to low level, high level to system requirement, including missing continuations.

#### REQ-TRAQ-SWL-41 Pandoc markdown rendering

Reqtraq SHALL invoke pandoc to convert markdown (and pandoc extensions like math, tables, and code) into HTML that correctly renders into generated reports.

##### Attributes:
- Parents:
- Rationale: pandoc is a markdown to HTML converter that has markdown-extensions for math, tables, and code.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-12 Top-down report

Reqtraq SHALL create a report mapping top-level system requirements down to child requirements and implementation.

##### Attributes:
- Parents: REQ-TRAQ-SWH-4
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-20 Filter top-down report

Reqtraq SHALL allow the user to filter the top down report by ID, title, body, attributes (parents, rationale, verification, safety impact).

##### Attributes:
- Parents: REQ-TRAQ-SWH-4, REQ-TRAQ-SWH-8
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-13 Bottom-up report

Reqtraq SHALL create a report mapping implementation up to child requirements and top-level system requirements.

##### Attributes:
- Parents: REQ-TRAQ-SWH-4
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-21 Filter bottom-up report

Reqtraq SHALL allow the user to filter the bottom up report by ID, title, body, attributes (parents, rationale, verification, safety impact).

##### Attributes:
- Parents: REQ-TRAQ-SWH-4, REQ-TRAQ-SWH-8
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-30 Issue report

Reqtraq SHALL allow the user to create a report showing issues with:
- Attributes - the attributes of each requirement are checked against the schema defined in the attributes.json
- References - parent attributes are checked to ensure they are pointing to valid requirements (they exist and are not deleted)

##### Attributes:
- Parents: REQ-TRAQ-SWH-14
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-31 Filter issue report

Reqtraq SHALL allow the user to filter the issue report.

##### Attributes:
- Parents: REQ-TRAQ-SWH-8, REQ-TRAQ-SWH-14
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-39 Output

Reqtraq report outputs SHALL be created as HTML.

##### Attributes:
- Parents:
- Rationale: The HTML can be used to create static HTML files and also to serve pages in the web app.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-19 Filtering of output

Reqtraq SHALL allow filtering by matching a regular expression against:
- requirement id
- requirement title
- requirement description/body

##### Attributes:
- Parents: REQ-TRAQ-SWH-4, REQ-TRAQ-SWH-8
- Rationale:
- Verification: Test
- Safety Impact: None

### req.go

Functions related to the handling of requirements and code tags.

The following types and associated methods are implemented:
- ReqGraph - The complete information about a set of requirements and associated code tags.
- Req - A requirement node in the graph of requirements.
- Schema - The information held in the schema file defining the rules that the requirement graph must follow.
- byPosition, byIDNumber and ByFilenameTag - Provides sort functions to order requirements or code.
- ReqFilter - The different parameters used to filter the requirements set.

**Data structure for keeping requirements and their hierarchy**

The interface between the parsing tool and the report generation tool is a data structure that maps requirement IDs to a requirement structure. The requirement structure holds all the data about the requirement that is needed for the report generation (ID, body, attributes, parents, children, etc.). The data structure is built by traversing the entire git repository and parsing:
- `.md` certification documents files that may define requirements,
- `.cc`, `.hh`, `.go` source files that may reference requirements.

#### REQ-TRAQ-SWL-1 Scan for markdown

Reqtraq SHALL scan all folders within a given path, relative to the git repository root, searching for markdown files which will then be parsed for requirements.

##### Attributes:
- Parents: REQ-TRAQ-SWH-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-27 Add requirements to graph

Reqtraq SHALL add any requirements, found by parsing the markdown, to the requirement graph.

##### Attributes:
- Parents: REQ-TRAQ-SWH-1, REQ-TRAQ-SWH-12
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-28 No repeated identifiers

Reqtraq SHALL ensure that requirement identifiers aren't repeated.

##### Attributes:
- Parents: REQ-TRAQ-SWH-12
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-10 Valid requirement references

Reqtraq SHALL check that the requirements referred to in each markdown document and source code file exists in the requirement graph.

##### Attributes:
- Parents: REQ-TRAQ-SWH-3
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-11 Invalid references to deleted requirements

Reqtraq SHALL ensure that links are not made to deleted requirements.

##### Attributes:
- Parents: REQ-TRAQ-SWH-3
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-22 Change history tracing

Reqtraq SHALL generate a list of all changelists that touched the definition or implementation of a given set of requirements, and the corresponding Problem Reports that these changelists belong to.

##### Attributes:
- Parents: REQ-TRAQ-SWH-9
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-29 Load schema

Reqtraq SHALL load a schema file 'attributes.json' which describes the valid range of requirement attributes and use it to validate requirements against.

##### Attributes:
- Parents: REQ-TRAQ-SWH-13
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-25 Uniform requirement ID format

Reqtraq SHALL check that the requirements defined in each document have a correct id, composed of four parts separated by `-`:

1. `REQ` or `ASM`
2. the project/system abbrev, identical to the first part of the document name where the requirement is defined, e.g. "TRAQ" for "TRAQ-100-ORD.md"
3. the requirement type:
    - `SYS` for system/overall requirements (defined in ORD documents)
    - `SWH` for software high-level requirements (defined in SRD documents)
    - `SWL` for software low-level requirements (defined in SDD documents)
    - `HWH` for hardware high-level requirements (defined in HRD documents)
    - `HWL` for hardware low-level requirements (defined in HDD documents)
4. a sequence number n such that requirements 1, 2, ..., n all exist, not necessarily in order

##### Attributes:
- Parents: REQ-TRAQ-SWH-12
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-26 ID allocation

Reqtraq SHALL check that given a requirement ID with sequence number N, all requirements with the same prefix and sequence numbers 0...N-1 exist and are defined in the current document (in any order).

##### Attributes:
- Parents: REQ-TRAQ-SWH-12
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-23 Deleted requirements

Reqtraq SHALL mark requirements whose title is prefixed with 'DELETED' as being deleted.

##### Attributes:
- Parents: REQ-TRAQ-SWH-10
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-45 Sort by position

Reqtraq SHALL include functions to sort a requirements list by position.

##### Attributes:
- Parents:
- Rationale: Used for providing easily readable output.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-46 Sort by ID

Reqtraq SHALL include functions to sort a requirements list by ID number.

##### Attributes:
- Parents:
- Rationale: Used for providing easily readable output.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-47 Sort by filename

Reqtraq SHALL include functions to sort a requirements list by filname tag.

##### Attributes:
- Parents:
- Rationale: Used for providing easily readable output.
- Verification: Test
- Safety Impact: None

### webapp.go

Functions for creating and servicing a web interface.

#### REQ-TRAQ-SWL-37 Web interface

Reqtraq SHALL support starting up a simple web interface for report generation. The syntax for starting up the web interface will be:

```
reqtraq web [:<port>]
```

The command must be executed in the repository for which the reports will be generated.

##### Attributes:
- Parents: REQ-TRAQ-SWH-17
- Rationale:
- Verification: Test
- Safety Impact: None
