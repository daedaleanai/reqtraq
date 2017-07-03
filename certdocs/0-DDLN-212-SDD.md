# Design Description for Reqtraq

Document Approval:
- Engineering, Program Manager: Luuk van Dijk
- Engineering, Engineer: Daniel Danciu
- Quality, Quality Engineer: Mukta Prasad

## Introduction

### Purpose

This document contains the definition of the software architecture and the <abbr title="Software Low-Level Requirement">SWL</abbr>s for the Reqtraq tool that will satisfy the <abbr title="Software High-Level Requirement">SWH</abbr>s specified in 0-DDLN-211-SRD. It follows Section 11.10 of DO-178C / ED-12C.

The purpose of the Reqtraq tool is to comply with the DO-178C / ED-12C traceability data requirements.

### Scope

This document discusses the following topics:
- Report generation
- Traceability of requirements data, change data
- Compliance with formatting
- Assumptions
- Interfacing

### Applicable Documents

#### External Documents

**RTCA DO-178C / EUROCAE ED-12C** Software Considerations in Air-borne Systems and Equipment Certification.

**Developing Safety-Critical Software: A Practical Guide for Aviation Software and DO-178C Compliance.** Rierson, Leanna. 2013

#### Internal Documents

**0-DDLN-6-SRS** Software Requirements Standards

*[TODO: SDS Software Design Standards]*

**0-DDLN-1-DS** Documentation Standards

**0-DDLN-107-CLSDD** Software Design Document Checklist

**0-DDLN-100-ORD** Overall Requirements Document for Reqtraq

**0-DDLN-211-SRD** Software Requirements Document for Reqtraq

### Definitions of Acronyms and Terms

#### Description of Terms

- Changelist: a git commit resulting in the creation, revision, or deletion of any version controlled material.

## Software Overview

### Inputs and Outputs

Data comes in from the requirements documents and code files in one or more Git repositories.

Reqtraq interacts with a Phabricator server using the Phabricator Conduit <abbr title="Application Programming Interface">API</abbr>.

Reqtraq has the following outputs:
- Traceability reports and issues, including hyperlinking between documents
- The next available requirement <abbr title="Identification">ID</abbr>

### Software Design and Implementation Details

Any report is generated from a `reqGraph` object which is a dictionary of requirements by IDs at a particular Git commit. A requirement is represented by an instance of the `Req` data structure.

## Low-level Software Requirements Identification

The SWLs of the system are as follows:

##### REQ-RQTQ-SWL-001 Requirements Storage

Requirements SHALL be stored in Lyx or Markdown files and version controlled by Git. Reqtraq is not responsible for the actual formatting or version control of each document. Instead Reqtraq leverages Git for storage and version control and Lyx/Latex or Markdown for formatting.

The git repository and location where each requirement document is stored is defined in 0-DDLN-10-DS.

Reqtraq will parse the requirement documents in the `certdocs` directory:
- Each requirement in the `.lyx` file is delimited by a Lyx `req:` note.
- Each heading in the `.md` file having a requirement id at the beginning represents the start of a requirement.

###### Attributes:
- Rationale: Git is the industry standard for version control. Lyx is the industry standard for formatting. Markdown is widespread and very easy to use.
- Parents: REQ-RQTQ-SWH-001
- Verification: Unit test
- Safety impact: None

##### REQ-RQTQ-SWL-014 Accessing and linking to requirements

Each time a change to a requirement document is committed, Reqtraq SHALL parse the document and alter the following information:

- each requirement described in the document, is wrapped into a **named anchor** so that the requirement can be directly linked to, e.g.

```
<a name="#REQ-RQTQ-SWL-001">REQ-RQTQ-SWL-001</a>
```

- each requirement referred to in the document is replaced with a link to that requirement. For example a reference to the High Level Requirement REQ-RQTQ-SYS-001 will be replaced with:

```
<a href="https://a/REQ-RQTQ-SYS-001">REQ-RQTQ-SYS-001</a>
```

Where `http://a` is the Daedalean URL redirector. The URL redirector will infer the name of the document where the requirement is defined (0-DDLN-0-SRD in our case) and use the Google Drive API or other methods to find the URL of the document, then redirect to it, e.g.:

```
<a href="https://doc-04-6g-docs.googleusercontent.com/....">REQ-RQTQ-SYS-001</a>
```

**TODO:** The URL in the example may be a Docs Url instead

Note that the URL redirector needs to infer the name of the document the requirement is defined in, as described in REQ-RQTQ-SWL-002.

###### Reqtraq triggering

Reqtraq SHALL have a Git server-hook component that automatically triggers each time a change to a requirement document is committed. Reqtraq will use the following rules to determine if a document is a requirement document

- the document is a `.lyx` or `.md` file

- the name of the document matches the naming conventions laid out in 0-DDLN-10-DS: [Project Num]-[Project Abbrev]-[Seq Num]-[Document Type Acronym]

###### Attributes:
- Rationale: this ensures that all documents defining or reference requirements don’t introduce typing mistakes.
- Parents: REQ-RQTQ-SWH-012
- Verification: Unit test
- Safety impact: None

##### REQ-RQTQ-SWL-002 Construct the requirement URL.

The <abbr title="Requirements Management Tool">RMT</abbr> SHALL infer the document where a requirement is defined solely based on the name of the requirement. This can be uniquely constructed as follows:

- strip the `REQ-` prefix from the requirement name

- take the section of the requirement name following `REQ-` (project/system abbrev), e.g. "RQTQ"

- append the sequence number for the document type (e.g. 100 for ORD, 212 for SDD, etc.)

- if the requirement is:
  - low level (SWL or HWL), append `-SDD`
  - high level (SWH or HWH), append `-SRD`
  - system level (SYS), append `-ORD`

###### Attributes:
- Rationale: simplicity and completeness: the ability to find a requirement only based on its name simplifies the development and the verification process.
- Parents: REQ-RQTQ-SWH-001
- Verification: Unit test
- Safety impact: None

##### REQ-RQTQ-SWL-003 Uniform requirement ID format.

The RMT SHALL check that the requirements defined in each document have a correct id:

- the first section of the requirement name is `REQ-`

- the next section (section 2) of the requirement is identical to the first section of the document (project/system abbrev), e.g. "RQTQ"

- section 3 of the requirement id is:

    - `SYS` for system/overall requirements (defined in ORD documents)

    - `SWH` for software high-level requirements (defined in SRD documents)

    - `SWL` for software low-level requirements (defined in SDD documents)

    - `HWH` for hardware high-level requirements (defined in HRD documents)

    - `HWL` for hardware low-level requirements (defined in HDD documents)

- section 4 of the requirement name is a sequence number n such that requirements 0, 1, ..., n-1 all exist, not necessarily in order

###### Attributes:
- Rationale: correct IDs are essential for tracing requirements.
- Verification: Unit test.
- Safety impact: None.
- Parent: REQ-RQTQ-SWH-002

##### REQ-RQTQ-SWL-004 Valid requirement references.

The RMT SHALL check that the requirements referred to in each document exist (and thus have a correct id):

- for each requirement reference, check if the referenced requirement exists in the requirement map constructed as described in REQ-RQTQ-SWL-015

###### Attributes:
- Rationale: invalid requirement references indicate an error in the requirement construction
- Verification: Unit test.
- Safety impact: None.
- Parent: REQ-RQTQ-SWH-002

##### REQ-RQTQ-SWL-005 ID allocation.

The RMT SHALL check that given a requirement ID with sequence number N, all requirements with the same prefix and sequence numbers 0...N-1 exist and are defined in the current document (in any order).

###### Attributes:
- Rationale: this helps ensure that no requirement sequence numbers are accidentally skipped.
- Verification: Unit test.
- Safety impact: None.
- Parent: REQ-RQTQ-SWH-003

##### REQ-RQTQ-SWL-017 Deleted requirements.

Deleted requirements are requirements that do not apply anymore (e.g. they are obsolete). Deleted requirements SHALL be marked by changing the title to "Deleted", for example "REQ-RQTQ-SWL-015 Deleted"

Deleted requirements SHALL not be checked for completeness and all the tasks associated with them SHALL be closed as WONTFIX. All references to a deleted requirement SHALL be marked as errors.

###### Attributes:
- Rationale: continuous requirement numbering helps ensure that no requirements were accidentally skipped and completely deleting requirements would create gaps in the numbering.
- Verification: Unit test.
- Safety impact: None.
- Parent: REQ-RQTQ-SWH-003

##### REQ-RQTQ-SWL-015 Data structure for keeping requirements and their hierarchy

The interface between the parsing tool and the report generation tool SHALL be a data structure that maps requirement IDs to a requirement structure. The requirement structure will hold all the data about the requirement that is needed for the report generation (ID, body, attributes, parents, children, etc.). The data structure is built by traversing the entire git repository and parsing all files that may contain or reference requirements, such as `.lyx`/`.md` requirement files and `.cc`/`.hh` source files).

###### Attributes:
- Rationale: this data structure will be used for report generation and graph verification.
- Verification: Unit test
- Safety impact: None
- Parents: REQ-RQTQ-SWH-004, REQ-RQTQ-SWH-005

##### REQ-RQTQ-SWL-020 Requirements document organization

Requirements documents SHALL be organized as follows:
  - requirements documents for a specific project SHALL be located in the `certdocs` top-level directory for that particular project
  - for projects that are part of a larger system, the system level requirements of the project will have parents in the system level requirements of the larger system
  - for projects that are self-contained, the system level requirements will have no parents

**Example 1: Self-contained project**
A project called VMU (Visual Motion Unit) is certified as a self-contained project. In this case, the top-level directory of the VMU project will contain a `certdocs` directory that contains the system level, high-level and low-level requirement documents:

```
vmu
  certdocs
     RQTQ-100-ORD.md
     RQTQ-211-SDD.md
     RQTQ-212-SDD.md
  vmu code files
```

The system requirements in `RQTQ-100-ORD.md` are top-level requirements, and will thus have no parents.

**Example 2: Project part of a larger system**
A project called VMU (Visual Motion Unit) is certified as a subsytem of a larger system (let's call it VG - visual guidance) that contains other related projects (LD - landmark detection, CD - cloud detector, etc.). In this case, the top-level directory of the VMU, the CD, and LD will each contain a `certdocs` directory that contains the system level, high-level and low-level requirement documents for their respective projects. The top-level directory for VG, the overall guidance system, will also contain a `certdocs` directory, with system requirements that are the parents of all system requirements defined in the VMU, LD or CD:

```
vg
  certdocs
     VG-100-ORD.md  # these are the top-level parent requirements
vmu
  certdocs
     VMU-100-ORD.md # these system requirements refer to parents in VG-100-ORD.md
     VMU-211-SDD.md
     VMU-212-SDD.md
  vmu code files
ld
  certdocs
     LD-100-ORD.md # also refer to parents in VG-100-ORD.md
     LD-211-SDD.md
     LD-212-SDD.md
  ld code files
cd
  ...
```
###### Attributes:
- Rationale: Reqtraq should support tracking requirements across sub-systems of a larger system.
- Parents: REQ-RQTQ-SWH-15, REQ-RQTQ-SWH-16
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-006 Tracing system to high, low level, implementation, test

Given a list of requirements, the RMT SHALL be able to generate parent and child requirements and code ordered from system to high level to low level requirement to implementation and test, including missing continuations.

###### Report structure

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

            - Changelists: list of differential requests implementing the requirement

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

**Note 3:** In the case of a system that has multiple projects (sub-systems), there will be more than one level of system requirements.

###### Attributes:
- Rationale: the report will be used by the certification authority to visualize requirement tracing.
- Parents: REQ-RQTQ-SWH-004, REQ-RQTQ-SWH-007
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-007. Tracing implementation, test to low, high, system level.

Given a list of requirements, the RMT SHALL be able to generate parent and child requirements and code, ordered from implementation or test to low level, high level to system requirement, including missing continuations.

**Note:** This is identical to REQ-RQTQ-SWL-006, except that the report is generated in reverse order.

###### Attributes:
- Rationale: the report will be used by the certification authority to visualize requirement tracing.
- Parents: REQ-RQTQ-SWH-005
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-008. Change impact tracing

The RMT SHALL be able to generate a list of all requirements changed between checked in versions of the project’s documentation, for use as input to the high-to-low and low-to-high tracing functions.

The report generation described in REQ-RQTQ-SWL-006 will be able to receive the following inputs:

- no input: in this case Reqtraq will generate a global HTML report listing all the requirements, from system to high level to low level, defined in the project associated with the repository (each project has its own Git repository)

- a list of requirement IDs (system, high level or low level): in this case the report will be generated for the given requirements, plus all their direct/indirect children. The direct/indirect parent requirements will also be listed, but all children other than the ones requested will be omitted.

- two git commit IDs (or git refs): in this case the report will contain all requirements that were changed between the given commits. If the 2nd commit id is missing, the current state of the repository is used.

Suggested usage (the directory in which reqtraq is run determines the project for which the report is generated):

```
reqtraq report 
reqtraq report REQ-RQTQ-SWL-008,REQ-RQTQ-SWL-009,REQ-RQTQ-SWL-010
reqtraq report d6cd1e2bd19e03a81132a23b2025920577f84e37
```

Note: a requirement is considered "changed" if either it was directly edited or one of the child requirements was edited.

###### Attributes:
- Rationale: the report will be used by the certification authority to visualize requirements that changed within a period of time.
- Parents: REQ-RQTQ-SWH-006
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-018 Phabricator export

The RMT SHALL export all requirements as Phabricator tasks, in the following format:

- Title: requirement title, e.g. "REQ-RQTQ-SWL-018 Phabricator export"
- Assigned To: empty
- Status: Open
- Priority: Normal
- Description: The requirement description
- Tags: Project code (e.g. RQTRQ in this case)
- Subscribers: empty
- Parents\: the parent requirements
- Children: the child requirements

If a task with the given requirement id already exists, then RMT will update the title, description and parents of the task, but all other fields will be left unchanged.

If a task’s title changes to "Deleted" it’s associated task and all its children will be marked as WONTFIX.

###### Attributes:
- Parents: REQ-RQTQ-SWH-006
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-009. Change history tracing

The RMT MUST be able to generate a list of all changelists that touched the definition or implementation of a given set of requirements, and the corresponding Problem Reports that these changelists belong to.

The report described in REQ-RQTQ-SWL-006 addresses this requirement. Each LLR will contain both the source files that implement it and the CLs that implemented it.

###### Attributes:
- Parents: REQ-RQTQ-SWH-007
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-010. Change justification tracing

The RMT SHALL verify and flag violations that changelists touching definitions or implementation of a requirement have a rationale-for-change field.

The current workflow forces each changelist to have a description and an associated task/problem report. The changelist description can be viewed as the rationale-for-change.

###### Attributes:
- Rationale: no changes should be allowed unless they were vetted and the justification accepted by an independent reviewer.
- Parents: REQ-RQTQ-SWH-008
- Verification: Test.
- Safety impact: None.

##### REQ-RQTQ-SWL-011. Report readability

The formats supported by RMT will be HTML and PDF, using the following syntax:

```
reqtraq report --format pdf
reqtraq report --format html
```

###### Attributes:
- Rationale: reports on the bidirectional traceability have to be submitted as evidence in certification trajectories. The report generated in REQ-RQTQ-SWL-006 addresses this issue.
- Parents: REQ-RQTQ-SWH-009
- Verification: Demonstration.
- Safety impact: None

##### REQ-RQTQ-SWL-012. Filtering of output

The report generation tool SHALL allow filtering by matching a regular expression against:

- requirement id
- requirement title
- requirement description/body

Suggested usage:
```
reqtraq report --title_filter="Motion estimation"
reqtraq report --id_filter="REQ-RQTQ-SWL-.*"
reqtraq report --body_filter="reprojection error"
```

###### Attributes:
- Rationale: useful in the development or verification phase to check a subset of requirements.
- Parents: REQ-RQTQ-SWH-010
- Verification: Demonstration
- Safety impact: None

##### REQ-RQTQ-SWL-016 Web interface

The RMT tool SHALL support starting up a simple web interface for report generation. The syntax for starting up the web interface will be:

```
reqtraq web [:<port>]
```

The command must be executed in the repository for which the reports will be generated.

###### Attributes:
- Rationale: useful for report navigation and easy report generation.
- Parents: REQ-RQTQ-SWH-013
- Verification: Demonstration
- Safety impact: None

##### REQ-RQTQ-SWL-013 Requirement attributes

The RMT SHALL be able to store a number of predefined attributes and enforce/flag mandatory/optional rules for them.

The attributes of each requirement MUST appear at the end of the requirement definition, one per line.

Attributes can be optional or mandatory. Each attribute has a name. Each attribute may have an associated regular expression to test for validity. Attributes are specified in an `attributes.json` file in the `certdocs` directory. For example, the attributes.json for the current document would be:

```
{ "attributes": [
  { "name": "Parent", "optional": false }, 
  { "name": "Verification", "value": "(Demonstration|Unit Test)", "optional": false },
  { "name": "Safety Impact", "optional": false } ] }
```

###### Attributes:
- Rationale: Attributes help define the importance of and the verification process for each requirement. Missing attributes indicate an incomplete requirement.
- Parents: REQ-RQTQ-SWH-011
- Verification: Demonstration
- Safety impact: None

##### REQ-RQTQ-SWL-019 Pandoc markdown rendering

The RMT tool SHALL invoke pandoc to convert markdown (and pandoc extensions like math, tables, and code) into HTML that correctly renders into generated reports.

* Requirements bodies can include code that is delimited with the triple-backtick delimiter (\```), which results in rendered HTML as follows:
```
int main(int argc, char** argv) {
        FlyAirplane();
}
```


* Requirements bodies can include math (both inline and display) that is delimited with the single dollar sign (\$) for inline, and the double dollar
sign (\$$) for display. They will be rendered in HTML reports using MathJax and look as follows:
    * Inline math: $x=y$
    * Display math:
$$
\frac{d}{dx}\left( \int_{0}^{x} f(u)\,du\right)=f(x).
$$

* Requirements bodies can include tables in any of the four pandoc table formats. An example simple table is shown here:

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




###### Attributes:
- Rationale: pandoc is a markdown to HTML converter that has markdown-extensions for math, tables, and code.
- Parents: REQ-RQTQ-SWH-014
- Verification: Demonstration
- Safety impact: None


### Other Assumptions

In the creation of these requirements it was assumed that Reqtraq users use Git for version control.
