# Software Requirements Document for Reqtraq

Document Approval:
- Engineering, Program Manager: Luuk van Dijk
- Engineering, Engineer: Daniel Danciu
- Quality, Quality Engineer: Anna Chernova

## Introduction

The <abbr title="Software High-Level">SWH</abbr> requirements for Reqtraq are
created based on the parent requirements in the <abbr title="Overall
Requirements Document">ORD</abbr>.

### Purpose

The purpose of this document is to define what the Reqtraq tool features should
do. The SWH requirements will be used for development of the <abbr
title="Software Low-Level">SWL</abbr> requirements document. It complies with
section 11.9 of DO-178C / ED-12C and Daedalean AG’s <abbr title="Software
Requirements Standards">SRS</abbr>.

The purpose of the Reqtraq tool is to comply with the DO-178C / ED-12C
traceability data requirements.

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

Reqtraq's goal is to ensure bidirectional traceability throughout all levels of
requirements and to identify faults in that traceability. In fulfilling these
goals, Reqtraq generates various reports and prevents the introduction of errors
by requiring compliance with the Reqtraq system.

### Functional Requirements

#### High-Level Requirements

The SWH requirements of the system are as follows:


##### REQ-TRAQ-SWH-1 Requirements discovery

The RMT SHALL find requirements that it manages in the controlled git repository
inside `.md` files.

###### Attributes:
- Rationale: Requirements must be change-controlled. Work done in the Git repository is tracked in a separate PR/ticket system, but all data that needs to be controlled shall be stored with the commits in Git.
- Parents: REQ-TRAQ-SYS-7
- Verification: Demonstration
- Safety impact: None


##### REQ-TRAQ-SWH-12 Requirements linking

The requirement IDs in the generated reports SHALL link to the requirement
definition.

###### Attributes:
- Rationale: Easy navigation of the requirement hierarchy is essential both for development and for verification.
- Parents: REQ-TRAQ-SYS-1
- Verification: Demonstration
- Safety impact: None


##### REQ-TRAQ-SWH-2 Requirement ID format

The RMT SHALL enforce a uniform requirement ID format. Defined to be:

REQ-[system abbreviation]-[requirement type]-[a sequential number]

e.g.: REQ-TRAQ-SWH-2

###### Attributes:
- Rationale: Tracing is only possible if requirements have unambiguous identifiers.
- Parents: REQ-TRAQ-SYS-1
- Verification: Unit test
- Safety impact: None


##### REQ-TRAQ-SWH-3 ID allocation

The RMT SHALL enforce that requirement numbers are allocated without repetition
or gaps.

###### Attributes:
- Rationale: It is easier to track no requirements have fallen through the cracks if the tools to manipulate them can check they are all enumerated.
- Parents: REQ-TRAQ-SYS-5
- Derived: Yes
- Verification: Unit test
- Safety impact: None


##### REQ-TRAQ-SWH-15 Multiple project systems

In order to combine systems into a larger product, the RMT SHALL be able to
trace System-Level Requirements to other, higher-level System-Level
Requirements.

###### Attributes:
- Rationale: In the development of broader systems, a hierarchy of system-level requirements might be necessary in order to segment the development into independent modules.  
- Parents: REQ-TRAQ-SYS-8
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-16 DELETED


##### REQ-TRAQ-SWH-4 Tracing top-down

The RMT SHALL be able to generate a report containing requirements, code
references, ordered and showing parenting relationship from system to high level
to low level requirement to implementation to test cases.

###### Attributes:
- Rationale: Needed for investigating everything related to an item.
- Parents: REQ-TRAQ-SYS-1
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-5 Tracing bottom-up

The RMT SHALL be able to generate a report containing requirements and code
references, ordered and showing parenting relationship from test cases to
implementation to low level to high level to system requirements.

###### Attributes:
- Rationale: Needed for investigating everything related to an item.
- Parents: REQ-TRAQ-SYS-1
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-6 Version differences

The RMT SHALL be able to generate a list of requirements at a specified version
which have been added to or changed from a previously checked in version,
wherein the updated material is traced bidirectionally.

###### Attributes:
- Rationale: According to DO-178C section 7.2.2.2, the certification authority must be able to see the changes that happened in a period of time, e.g. between two successive audits.
- Parents: REQ-TRAQ-SYS-1
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-22 Implementation association with LLR(s) or HLR(s)

The RMT SHALL enforce that all source code, test cases and test procedures
reference LLR(s) or HLR(s).

###### Attributes:
- Rationale: The entire implementation should be traced to requirements.
- Parents: REQ-TRAQ-SYS-1
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-7 Change history

The RMT SHALL be able to report all the Git commits that touched the definition
of a given set of requirements. The corresponding tasks and Problem Reports
causing these commits SHALL also be included.

###### Attributes:
- Rationale: The certification authority must be able to verify that the implementation of each requirement followed the appropriate process.
- Parents: REQ-TRAQ-SYS-3
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-8 DELETED


##### REQ-TRAQ-SWH-9 Output readability

The output from this tool SHALL be available in a pretty-printed form.

###### Attributes:
- Rationale: reports on the bidirectional traceability have to be submitted as evidence in certification trajectories.
- Parents: REQ-TRAQ-SYS-1
- Verification: Demonstration
- Safety impact: None


##### REQ-TRAQ-SWH-10 Output filtering

The requirement sets of all the above outputs SHALL be filterable by attributes
such as ID, title or body.

###### Attributes:
- Rationale: to be of use in daily routine, the tool should allow quick querying by developers.
- Parents: REQ-TRAQ-SYS-1
- Derived: Yes
- Verification: Demonstration
- Safety impact: None


##### REQ-TRAQ-SWH-11 Attribute verification

The RMT SHALL be able to verify whether the appropriate requirement-level
attribute fields are correctly completed and whether the mandatory ones are
present.

###### Attributes:
- Rationale: Attributes help define the importance of and the verification process for each requirement. Missing attributes indicate an incomplete requirement.
- Parents: REQ-TRAQ-SYS-6
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-21 Safety impact inheritance

The RMT SHALL enforce the "Safety impact" attribute of parent requirements is
correctly propagated to the children requirements.

###### Attributes:
- Rationale: This ensures the safety requirements, as identified at the system level, are properly implemented.
- Parents: REQ-TRAQ-SYS-6
- Verification: Test
- Safety impact: None


##### REQ-TRAQ-SWH-17 DELETED


##### REQ-TRAQ-SWH-18 DELETED


##### REQ-TRAQ-SWH-19 DELETED


### Software Interfaces Requirements


##### REQ-TRAQ-SWH-13 Web interface

The report generation tool SHALL have a simple web interface that allows
generation and filtering of reports.

###### Attributes:
- Rationale: Since the reports we generate are easier to navigate online, a web interface will allow generating and navigating reports on the fly. It also opens up report generation and verification to people who are less comfortable with a command line tool.
- Parents: REQ-TRAQ-SYS-5
- Derived: Yes
- Verification: Demonstration
- Safety impact: None


##### REQ-TRAQ-SWH-14 Requirement Rich Formatting

The RMT SHALL allow for us to express rich markdown concepts in requirements
descriptions, e.g. math, tables, and code.

###### Attributes:
- Rationale: Technical Requiments often need to document equations, tables of data, or code.
- Parents: REQ-TRAQ-SYS-6
- Verification: Demonstration
- Safety impact: None


##### REQ-TRAQ-SWH-20 DELETED


### Other Assumptions

In the creation of these requirements it was assumed that Reqtraq users use Git
for version control.
