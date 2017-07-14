# Software Requirements Document for Reqtraq

Document Approval:
- Engineering, Program Manager: Luuk van Dijk
- Engineering, Engineer: Daniel Danciu
- Quality, Quality Engineer: Mukta Prasad

## Introduction

The <abbr title="Software High-Level">SWH</abbr> requirements for Reqtraq are created based on the parent requirements in the <abbr title="Overall Requirements Document">ORD</abbr>.

### Purpose

The purpose of this document is to define what the Reqtraq tool features should do. The SWH requirements will be used for development of the <abbr title="Software Low-Level">SWL</abbr> requirements document. It complies with section 11.9 of DO-178C / ED-12C and Daedalean AG’s <abbr title="Software Requirements Standards">SRS</abbr>.

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

**FAA Order 8110.49** Software Approval Guidelines. 2003.

**RTCA DO-178C / EUROCAE ED-12C** Software Considerations in Air-borne Systems and Equipment Certification.

**IEEE 830-1998 Appendix A** Recommended Practice for Software Requirements Specifications. IEEE Standards Association. 2009.

**Developing Safety-Critical Software: A Practical Guide for Aviation Software and DO-178C Compliance.** Rierson, Leanna. 2013

#### Internal Documents

**DDLN-6-SRS** Software Requirements Standards

**DDLN-1-DS** Documentation Standards

**DDLN-100-ORD** Overall Requirements Document for Reqtraq

### Definitions of Acronyms and Terms

#### Description of Terms

- Changelist: a git commit resulting in the creation, revision, or deletion of any version controlled material.
- Problem report: a change request identifying a missing, non-functioning, or non-compliant feature.

## Software Overview

### Software Goals

Reqtraq's goal is to ensure bidirectional traceability throughout all levels of requirements and to identify faults in that traceability. In fulfilling these goals, Reqtraq generates various reports and prevents the introduction of errors by requiring compliance with the Reqtraq system.

### Functional Requirements

#### High-Level Requirements

The SWH requirements of the system are as follows:

##### REQ-TRAQ-SWH-1 Requirements Storage

The RMT SHALL persistently store and retrieve requirements that it manages and their change history in the controlled document repository in the form of `.md` files.

###### Attributes:
- Rationale: requirements must be change-controlled. We do this in Git repositories. The RMT must use this and only this to store the requirements. Work done in Git repositories is tracked in a separate PR/ticket system, but all data that needs to be controlled shall be stored with the commits in Git.
- Parents: REQ-TRAQ-SYS-1, REQ-TRAQ-SYS-2, REQ-TRAQ-SYS-3, REQ-TRAQ-SYS-4
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-12 Requirements Linking

Given a requirement ID, one should be able to easily navigate to the requirement definition without any extra information. That is, the requirement ID should uniquely identify the file where the requirement is defined. As a consequence, Reqtraq will be able to construct a link to any requirement referenced in a document.

###### Attributes:
- Rationale: easy navigation of the requirement hierarchy is essential both for development and for verification.
- Parents: REQ-TRAQ-SYS-1, REQ-TRAQ-SYS-7
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-2 Uniform requirement ID format

The RMT SHALL enforce a uniform requirement ID format. Defined to be:

REQ-[project/system abbreviation]-[requirement type identifier]-[a unique alphanumeric sequence]

e.g.: REQ-TRAQ-SWH-2

There is no special notation for derived requirements.

###### Attributes:
- Rationale: Tracing is only possible if requirements have unambiguous identifiers.
- Parents: REQ-TRAQ-SYS-1
- Verification: Unit test
- Safety impact: None

##### REQ-TRAQ-SWH-3 ID allocation

The RMT SHALL enforce that requirement numbers are allocated without repetition or gaps. Deleted requirements are replaced by placeholders.

###### Attributes:
- Rationale: It is easier to track no requirements have fallen through the cracks if the tools to manipulate them can check they are all enumerated.
- Parents: REQ-TRAQ-SYS-5
- Verification: Unit test
- Safety impact: None

##### REQ-TRAQ-SWH-15 Multiple project systems
In order to combine systems into a larger product, the RMT SHALL be able to trace System-Level Requirements to other, higher-level System-Level Requirements.

###### Attributes:
- Rationale: In the development of broader systems, a hierarchy of system-level requirements might be necessary in order to segment the development into independent modules.  
- Parents: REQ-TRAQ-SYS-8
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-16 Self-contained projects
In the instance of self-contained projects or systems, System-Level Requirements SHALL NOT have parent requirements in external documents.

###### Attributes:
- Rationale: In the development of self-contained systems, all requirements found in all documents must have full traceability. Therefore in the instance only one project within a larger system is being developed, no system-level requirements should have parents.
- Parents: REQ-TRAQ-SYS-8
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-4 Tracing system to high, low level, implementation, test

The RMT SHALL, given a list of requirements given to or generated by the project as checked in to the repository, be able to generate parent and child requirements and code ordered from system to high level to low level requirement to implementation and test, including missing continuations.

###### Attributes:
- Rationale: an incomplete requirement graph indicates that some requirements were not fulfilled.
- Parents: REQ-TRAQ-SYS-1, REQ-TRAQ-SWH-7
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-5 Tracing implementation, test to low, high, system level

The RMT SHALL, given a list of requirements given to or generated by the project as checked in to the repository, be able to generate lists of parent and child requirements and code, ordered from implementation or test to low level, high level to system requirement, including missing continuations.

###### Attributes:
- Rationale: an incomplete requirement graph indicates that some requirements were not fulfilled.
- Parents: REQ-TRAQ-SYS-1, REQ-TRAQ-SWH-7
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-6 Change impact tracing

The RMT SHALL be able to generate a list of all requirements changed between checked in versions of the project’s documentation, for use as input to the high-to-low and low-to-high tracing functions.

###### Attributes:
- Rationale: certification authority must be able to see the changes that happened in a period of time, e.g. between two successive audits.
- Parents: REQ-TRAQ-SYS-2
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-7 Change history tracing

The RMT SHALL be able to generate a list of all changelists that touched the definition or implementation of a given set of requirements, and the corresponding lists of Problem Reports that these changelists belong to.

###### Attributes:
- Rationale: certification authority must be able to verify that the implementation of each requirement followed the appropriate process.
- Parents: REQ-TRAQ-SYS-3
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-8 Change justification tracing

The RMT SHALL verify and flag violations that changelists touching definitions or implementation of a requirement have a rationale-for-change field.

###### Attributes:
- Rationale: no changes should be allowed unless they were vetted and the justification accepted by an independent reviewer.
- Parents: REQ-TRAQ-SYS-3
- Verification: Test
- Safety impact: None

##### REQ-TRAQ-SWH-9 Output readability

The output from this tool SHALL be available in a pretty printable form.

###### Attributes:
- Rationale: reports on the bidirectional traceability have to be submitted as evidence in certification trajectories.
- Parents: REQ-TRAQ-SYS-6
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-10 Output filtering

The requirement sets of all the above outputs SHALL be filterable by regular expressions on the contents.

###### Attributes:
- Rationale: to be of use in daily routine, the tool should allow quick querying by developers.
- Parents: REQ-TRAQ-SYS-5
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-11 Attribute storage and verification

The RMT SHALL be able to store a number of predefined attributes and enforce/flag mandatory/optional rules for them.

###### Attributes:
- Rationale: Attributes help define the importance of and the verification process for each requirement. Missing attributes indicate an incomplete requirement.
- Parents: REQ-TRAQ-SYS-6
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-17 External Parent Requirements

The RMT SHALL be able to store optional, non-traced External Parent Requirements.

###### Attributes:
- Rationale: Large organisations may have compartmentalised divisions that follow different requirement tracing protocols. This allows for collaboration in these situations.
- Parents: REQ-TRAQ-SYS-9
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-18 External Parent Requirement Coverage Analysis

If any individual System-Level Requirement has an optional, external parent requirement, then all System-Level Requirements in that document SHALL also have either a parent within the document or an external parent.

###### Attributes:
- Rationale: Unbroken traceability requires that all requirements are traced to the highest possible requirement document. In the case of an optional, higher level parent this attribute does not always need to be present, but when it is, all other requirements in that document will also have a parent requirement that needs to be traced.  
- Parents: REQ-TRAQ-SYS-9
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-19 External Parent Requirement Storage

External Parent Requirements, if they exist, SHALL be stored in a separate repository from the traceable requirements.

###### Attributes:
- Rationale: ReqTraq will not parse the repository for this type of requirement, and if such a type is found they will be flagged as an error.  
- Parents: REQ-TRAQ-SYS-9
- Verification: Demonstration
- Safety impact: None

### Software Interfaces Requirements

##### REQ-TRAQ-SWH-13 Web interface

The report generation tool SHALL have a simple web interface that allows generation and filtering of reports.

###### Attributes:
- Rationale: since the reports we generate are easier to navigate online, a web interface will allow generating and navigating reports on the fly. It also opens up report generation and verification to people who are less comfortable with a command line tool.
- Parents: REQ-TRAQ-SYS-5
- Verification: Demonstration
- Safety impact: None

##### REQ-TRAQ-SWH-14 Requirement Rich Formatting

The RMT SHALL allow for us to express rich markdown concepts in requirements descriptions, e.g. math, tables, and code.

###### Attributes:
- Rationale: Technical Requiments often need to document equations, tables of data, or code.
- Parents: REQ-TRAQ-SYS-6
- Verification: Demonstration
- Safety impact: None

### Other Assumptions

In the creation of these requirements it was assumed that Reqtraq users use Git for version control.
