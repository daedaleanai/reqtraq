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

Reqtraq uses a configuration file to determine:
- What documents are part of the requirements identification. Reqtraq will parse all certification documents listed in the configuration and extract requirements from all of them.
- What kind of requirements can be found in each of these documents. Reqtraq must categorize requirements according to their level and prefix.
- What are the relationships between documents and requirements. Requirements at different levels have a parent-children relationship that is specified by the reqtraq configuration.
- Where the implementation of low-level-requirements is located. Reqtraq must check all implentation and tests in order to complete its traceability report.
- What attributes are allowed by the reqtraq schema and their allowed range of values. This may depend on the document or requirement level.

### Software Design and Implementation Details

Documents (markdown and source code) are discovered and parsed from the current Git repository. Each requirement found is parsed into an instance of the `Req` data structure, and each code function in the `Code` data structure. They are all held within a `ReqGraph` object, which is a dictionary of requirements by IDs at a particular Git commit.

Various actions, such as comparison or filtering, are then performed on the `ReqGraph` object, depending on the type of output requested. The requested output is then generated.

ReqGraph source code is arranged as follows:
- main.go: The main entry point to the program, defines the top level command handler and invokes one of the commands defined in:
    - `completion_cmd.go`: Defines a `completion` subcommand that prints completion scripts for multiple shells (bash, zsh and fish).
    - `list_cmd.go`: Defines a `list` subcommand that lists all requirements in the given certdoc.
    - `nextid_cmd.go`: Defines a `nextid` subcommand that prints the next requirement id for the given certdoc.
    - `report_cmd.go`: Defines a `report` subcommand that creates HTM reports.
    - `validate_cmd.go`: Defines a `validate` subcommand that runs the validation checks on all certification documents.
    - `web_cmd.go`: Defines a `web` subcommand that runs the web application.
- req.go: The top-level functions dealing with finding and discovering markdown and source code files
    - parsing.go: Reading and parsing markdown files
    - code.go: Reading and parsing source code files. Reqtraq can use ctags or optionally libclang to obtain code references.
    - clang.go: Parsing the AST using libclang and collecting references to implementation and tests.
- diff.go: Comparison of ReqGraph objects
- report.go: Generating html reports to save to disk or provide to a web server
- matrices.go: Generating traceability tables to provide to a web server
- webapp.go: Launch and service a local web server
- repos/repos.go: Keeps a registry of all repositories where code and certification documents can be found
- linepipes/run.go: Wrapper functions the golang command interface
- config/config.go: Parses the reqtraq configuration for the git repository in the current directory. 
Registers any parent and children repositories found in the configuration file, and recursively parses their configuration.

## Low-level Software Requirements Identification

### code.go

Functions which deal with source code files. Source code is discovered within a given path and searched for functions and associated requirement IDs. The external program Universal Ctags is used to scan for functions.

#### REQ-TRAQ-SWL-6 Check for supported source code

The code component SHALL check the provided source code files and make sure only supported files are parsed as code.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-7 DELETED

#### REQ-TRAQ-SWL-8 Ctags

Reqtraq SHALL use the Universal Ctags application to parse supported source code files (those with an extension: c, h, cc, hh or go) for function names, and store them.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-9 Requirement IDs

Reqtraq SHALL scan source code containing functions for comments which contain requirement IDs preceding the functions, and store them.

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

#### REQ-TRAQ-SWL-65 Code parsers

Reqtraq SHALL provide functionality to register code parsers during runtime. Ctags is built-in, 
but other parsers can optionally be compiled-in and registered during initialization.

##### Attributes:
- Parents: 
- Rationale: To avoid dependencies on libclang for users that don't need libclang.
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

### repos/repos.go

Keeps a registry of all repositories where code and certification documents can be found. Reqtraq interacts with multiple repositories where the requirements are defined.
The repos module ensures that the correct instance of each repository is used and provides facilities for identifying repositories based on names, as well as overriding or registering repositories. It wraps git commands to checkout specific revisions of each repository.

#### REQ-TRAQ-SWL-16 Wrap git commands

Reqtraq SHALL wrap git commands needed with go functions.

##### Attributes:
- Parents: REQ-TRAQ-SWH-7
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-49 Repository registry

Reqtraq SHALL keep a repository registry that provides access to multiple repositories and their contents by name.

##### Attributes:
- Parents: REQ-TRAQ-SWH-18
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-50 Repository override

Reqtraq SHALL provide facilities for overriding repositories in order to check different revisions or the differences between multiple revisions.

##### Attributes:
- Parents: REQ-TRAQ-SWH-18
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-51 Find files

Reqtraq SHALL provide facilities for finding files with matching patterns within a repository.

##### Attributes:
- Parents: REQ-TRAQ-SWH-18, REQ-TRAQ-SWH-2
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

#### REQ-TRAQ-SWL-59 CLI entry point

reqtraq SHALL implement a root command which will act as a CLI entrypoint and which will contain all other subcommands.

##### Attributes:
- Parents: REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-60 Setup configuration

reqtraq SHALL implement a configuration setup that ensures the global configuration is configured 
before any command tries to access it.

##### Attributes:
- Parents:
- Rationale: To ensure that the configuration is loaded for all commands.
- Verification: Test
- Safety Impact: None

### parsing.go

Functions for parsing requirements out of markdown documents.

The entry point is ParseMarkdown which in turns calls other functions as follows:
- ParseMarkdown: Scans file one line at a time looking for requirements that either formatted within ATX headings
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

Attributes can be optional or mandatory. Each attribute has a name. Each attribute may have an associated regular expression to test for validity. Attributes are specified in the configuration file `reqtraq_config.json` file in the root of the repository. For more information about the format of the configuration file see the [Config](#config/config.go) section.
Attributes are specified per document or globally as common attributes that apply to any requirements found in any document.

Reqtraq reads the attributes of each requirement held in a requirement table from each column of the table. The first row of the table contains the attribute name for each column, the first column being "ID" to represent requirement ID. The second row is a delimiter. The third row onward contains the requirement ID and associated attribute text as shown:

```
> | ID | Title | Body | Attribute1 | Attribute2 |
> | --- | --- | --- | --- | --- |
> | ReqID1 | <text> | <text> | <text> | <text> |
> | ReqID2 | <text> | <text> | <text> | <text> |
```

As with ATX headings, attributes can be optional or mandatory as specified in the `reqtraq_config.json` file.

#### REQ-TRAQ-SWL-24 DELETED

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
- Attributes - the attributes of each requirement are checked against the schema defined in the reqtraq_config.json
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

#### REQ-TRAQ-SWL-1 Parse all requirement documents

Reqtraq SHALL parse all requirements found in any of the certification documents provided by the configuration component.

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

#### REQ-TRAQ-SWL-29 DELETED

#### REQ-TRAQ-SWL-25 Uniform requirement ID format

Reqtraq SHALL check that the requirements defined in each document have a correct id, composed of four parts separated by `-`:

1. The variant of the requirement, which must be either `REQ` or `ASM`
2. the project/system prefix, which is specified by the document as part of the requirement specification
3. the requirement level, which is specified by the document as part of the requirement specification. Typically it is one of:
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

#### REQ-TRAQ-SWL-17 Checkout alternative version

Reqtraq SHALL create a temporary version of an old version of the requirements documents to allow them to be loaded.

##### Attributes:
- Parents: REQ-TRAQ-SWH-8
- Rationale:
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

### config/config.go

Reqtraq contains a configuration component that parses an arbitrary number of configuration files named `reqtraq_config.json` to determine the 
structure of the project and requirements. Requirements and implementation can be scattered across multiple repositories and reqtraq must
be able to locate all dependencies and collect requirement and code information from all of them.

When reqtraq is started, the configuration from the Git repository in the current directory is parsed. A `reqtraq_config.json` file is located at
the root of the repository and contains information regarding:
- The name of the repository.
- Parent and children repositories that must be checked by reqtraq.
- A list of certification documents defined in the current repository
- A list of common requirement attributes across all documents and repositories.

A sample configuration file is shown below:

```json
{
    "repoName": "projectB",
    "parentRepository": {
        "repoName": "projectA",
        "repoUrl": "https://github.com/org/projectA"
    },
    "childrenRepositories": [
        {
            "repoName": "projectC",
            "repoUrl": "https://github.com/org/projectC"
        }
    ],
    "commonAttributes": [
        {
            "name": "Rationale",
            "required": "any"
        },
        {
            "name": "Verification",
            "value": "(Demonstration|Unit [Tt]est|[Tt]est)"
        }
    ],
    "documents": [
        {
            "path": "TEST-138-SDD.md",
            "prefix": "TEST",
            "level": "SWL",
            "parent": {
                "prefix": "TEST",
                "level": "SWH"
            },
            "attributes": [
                {
                    "name": "Custom Attribute",
                    "required": "false",
                },
            ],
            "asmAttributes": [
                {
                    "name": "Validation",
                    "required": "true",
                },
            ],
            "implementation": {
                "code": {
                    "paths": ["code"],
                    "matchingPattern": ".*\\.(cc|hh)$",
                    "ignoredPatterns": [".*_ignored\\.(cc|hh)$"]
                },
                "tests": {
                    "paths": ["test"],
                    "matchingPattern": ".*_test\\.(cc|hh)$"
                }
                "codeParser": "clang",
                "compilationDatabase": "path/to/compile_commands.json",
                "compilerArguments": ["-Os", "-Iinclude"]
            }
        }
    ],
}
```

Parent and children repositories are specified via the name of the repository and their associated URL.
There can only be one parent repository per configuration file, but multiple children repositories can exist.

Each document contains:
- A path where it can be located inside the repository that defines it.
- A requirement specification in terms of a `prefix` and `level`. Together they fully specify how the 
requirements in the document will look like. A document with level `SWL` and prefix `TEST` will have 
requirements of the form `REQ-TEST-SWL-1`.
- An optional parent. If there is a parent document, it contains also a `prefix` and `level` with the 
specification of the parent so that requirements can be linked from children to parent. The parent also
implies a `Parents` attribute with a filter for the requirement specification of the parent will be 
added to the schema of the document.
- An implementation, which consists of separated matchers for both code and tests. It accepts the 
following parameters:
  - `codeParser`: The selected code parser. Defaults to `ctags`. Available values are `ctags` and `clang`.
  - `compilationDatabase`: A path to a clang compilation database where the compiler invokation for 
  each translation unit can be found.
  - `compilerArguments`: If the source file is not found in the compilation database (which can happen for 
  header files, since they are not a translation unit), it will fall back to the arguments specified 
  in this list.
  each translation unit can be found.
- Any attributes for its requirements and assumptions.

Common attributes are appended to the attribute schema in each document to fully define the schema 
of the attributes in the requirements that belong to the document.

Parsing the configuration is an involved process that requires:
- Finding the top-level repository that does not have any parents.
- Iterating all the children from the top repository and parsing all requirement documents.

The result of this process is a structure that contains, for each repository, all the certification 
documents that must be parsed. This is used by other components (e.g.: the req.go component) to parse 
and validate requirements.

#### REQ-TRAQ-SWL-52 Find top-level configuration

Reqtraq SHALL find the top level configuration file and parse the complete configuration starting from 
the top of the configuration tree.

##### Attributes:
- Parents: REQ-TRAQ-SWH-18, REQ-TRAQ-SWH-13
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-53 Parse `reqtraq_config.json`

Reqtraq SHALL provide a method to parse a `requirement_config.json` file, validating them 
(making sure they contain all relevant information and doesn't contradict other files)

##### Attributes:
- Parents: REQ-TRAQ-SWH-15
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-54 Find certification document

Reqtraq SHALL provide a method to find a certification document inside any of the associated repositories.

##### Attributes:
- Parents: REQ-TRAQ-SWH-18
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-55 Find linked documents

Reqtraq SHALL provide a method to find linked requirements across different documents that share a 
parent/child relationship.

##### Attributes:
- Parents: REQ-TRAQ-SWH-15
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-56 Find code and test files

Reqtraq SHALL provide a method to find code and test files in each associated document.
The configuration shall allow to specify folders where code and tests are located, as well as
positive and negative filtering criteria for the files within those folders.

##### Attributes:
- Parents: REQ-TRAQ-SWH-18
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-64 Libclang arguments

Reqtraq SHALL provide options in its configuration file to obtain a compilation database and clang 
arguments required for parsing code using libclang.

##### Attributes:
- Parents: 
- Rationale: In order to effectively parse code where the AST construction depends on the build 
arguments and included files.
- Verification: Test
- Safety Impact: None

### completion_cmd.go

The completion command takes advantage of the underlying cobra infrastructure to print completion 
scripts for different shells. Supported shells are `bash`, `zsh` and `fish`.

#### REQ-TRAQ-SWL-57 Generate shell completions

Reqtraq SHALL provide a subcommand to generate shell completions.

##### Attributes:
- Parents: REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

### list_cmd.go

The list command implements the CLI for listing all requirements in a given certification document

#### REQ-TRAQ-SWL-33 CLI list requirements

Reqtraq SHALL provide a command line option to list the requirements within a given requirements document.

##### Attributes:
- Parents: REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

### nextid_cmd.go

The nextid command implements the CLI for displaying the next requirement ID in a given certification document.

#### REQ-TRAQ-SWL-34 CLI next ID

Reqtraq SHALL provide a command line option to provide the user with the next available ID in a given requirements document.

##### Attributes:
- Parents: REQ-TRAQ-SWH-12, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

### report_cmd.go

The report command implements the CLI for creating HTML reports with requirement trees or issues.

#### REQ-TRAQ-SWL-35 CLI reports

Reqtraq SHALL provide a command line options to generate reports showing top-down or bottom-up views on the requirements graph.

##### Attributes:
- Parents: REQ-TRAQ-SWH-4, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

### validate_cmd.go

The validate command implements the CLI for validating requirement graphs.

#### REQ-TRAQ-SWL-36 CLI validate

Reqtraq SHALL provide a command line option to validate a requirements graph and output to the terminal or to a report.

##### Attributes:
- Parents: REQ-TRAQ-SWH-14, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-66 CLI validate json output

Reqtraq SHALL allow the user to obtain a json file with the list of issues in the base repository 
compatible with Phabricator.

##### Attributes:
- Parents: REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

### web_cmd.go

The validate command implements the CLI for validating requirement graphs.

#### REQ-TRAQ-SWL-58 CLI web

Reqtraq SHALL provide a command line option to start a local web sever to generate reports, and list issues.

##### Attributes:
- Parents: REQ-TRAQ-SWH-14, REQ-TRAQ-SWH-16
- Rationale:
- Verification: Test
- Safety Impact: None

### clang.go

Reqtraq can use libclang to parse the ast of any linked code and obtain code references. This option
is preferred for a fine-grained code parsing, while ctags is preferred for compatibility with other 
languages like Go.

#### REQ-TRAQ-SWL-61 Support for libclang backend to parse code

Reqtraq SHALL provide libclang as a backend to parse code and find public functions/methods and type 
aliases.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-62 Skip anonymous and detail namespaces

Reqtraq SHALL ensure that code inside anonymous and detail namespaces is not reported for 
requirements tracking.

##### Attributes:
- Parents: 
- Rationale: It belongs only as an implementation detail that is needed, but not functionally directly related 
to requirements.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWL-63 Skip private methods

Reqtraq SHALL ensure that code inside a non-public path is not reported for requiremenstr tracking.
requirements tracking.

##### Attributes:
- Parents: 
- Rationale: It belongs only as an implementation detail that is needed, but not functionally directly related 
to requirements.
- Verification: Test
- Safety Impact: None

## Appendix

### Deleted Requirements

#### DELETED-7 Skip testdata

Reqtraq SHALL skip any folder named "testdata" when searching for source code.

##### Attributes:
- Parents: REQ-TRAQ-SWH-2
- Rationale: testdata folder may contain source code, which should not be included in the traceability.
- Verification: Test
- Safety Impact: None

#### DELETED-24 Certification document names

Reqtraq SHALL check the validity of a certification document name by checking the parts delimited by `-`:

1. Project abbreviation, which shall be the same for all the certification documents of a system, e.g. "TRAQ"
2. Document type sequence number, e.g. "138"
3. Document type, e.g. "SDD"

##### Attributes:
- Parents: REQ-TRAQ-SWH-11
- Rationale:
- Verification: Test
- Safety Impact: None

#### DELETED-29 Load schema

Reqtraq SHALL load a schema file 'attributes.json' which describes the valid range of requirement attributes and use it to validate requirements against. The schema allows the following settings to be specified:
- name: the name of attribute expected
- filter: which requirements the attribute applies to, as a regexp
- required: has one of the following values:
    - true: the attribute is required (default value if not set)
    - false: the attribute is optional
    - any: at least one of the attributes marked as this is required
- value: the value expected for the attribute, as a regexp

##### Attributes:
- Parents: REQ-TRAQ-SWH-13
- Rationale:
- Verification: Test
- Safety Impact: None
