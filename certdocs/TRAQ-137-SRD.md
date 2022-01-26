# Software Requirements Document for Reqtraq

## Introduction

The software <abbr title="High-Level Requirement">HLR</abbr>s for Reqtraq are created based on the parent requirements in the <abbr title="Overall Requirements Document">ORD</abbr>.

### Purpose

The purpose of this document is to define what the Reqtraq tool features should do. The HLRs will be used for development of the software <abbr title="Low-Level Requirement">LLR</abbr>s document. It complies with section 11.9 of DO-178C / ED-12C and Daedalean AG’s <abbr title="Requirements Standards">RS</abbr>.

The purpose of the Reqtraq tool is to comply with the DO-178C / ED-12C traceability data requirements.

### Scope

This document defines the software requirements for the Reqtraq tool, namely what the tool should do in order to meet the system requirements.

### Applicable Documents

#### External Documents

**RTCA DO-178C / EUROCAE ED-12C** Software Considerations in Air-borne Systems and Equipment Certification.

#### Internal Documents

**DDLN-1-DS** Documentation Standards

**DDLN-6-RS** Requirements Standards

**TRAQ-100-ORD** Overall Requirements Document for Reqtraq

### Nomenclature and Description of Terms

#### Description of Terms

N/A

## Reqtraq Software Overview

### Software Goals

Reqtraq's goal is to ensure bidirectional traceability throughout all levels of requirements and to identify faults in that traceability. In fulfilling these goals, Reqtraq generates various reports and prevents the introduction of errors by requiring compliance with the Reqtraq system.

Reqtraq should be as lightweight as possible and allow for human and machine interaction.

## High-level Software Requirements Identification

### Functional Requirements

#### REQ-TRAQ-SWH-1 Document Discovery

Reqtraq SHALL discover requirements within markdown files in the repository and add them to the requirements graph.

##### Attributes:
- Parents: REQ-TRAQ-SYS-1
- Rationale: Rich text allows for more meaningful requirement definition.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-2 Source Code Discovery

Reqtraq SHALL discover source code functions within compatible source language files in the repository and add them to the requirements graph.

##### Attributes:
- Parents: REQ-TRAQ-SYS-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-3 Link Validation

Reqtraq SHALL validate tracing between requirements, implementation and tests; Links must be to parent level and cannot be to deleted requirements.

##### Attributes:
- Parents: REQ-TRAQ-SYS-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-4 Traceability Reports

Reqtraq SHALL allow the user to view reports showing a top-down or bottom-up view of the requirements graph.

##### Attributes:
- Parents: REQ-TRAQ-SYS-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-5 Traceability Tables

Reqtraq SHALL allow the user to view tables showing the tracing between different levels of requirements and the source code.

##### Attributes:
- Parents: REQ-TRAQ-SYS-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-6 Safety Impact

Reqtraq SHALL validate that requirements marked as having a safety impact have the property flowed down to child requirements.

##### Attributes:
- Parents: REQ-TRAQ-SYS-1
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-7 Git Repositories

Reqtraq SHALL use git repositories to store and track changes to requirements documents.

##### Attributes:
- Parents: REQ-TRAQ-SYS-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-8 Requirement Comparison

Reqtraq SHALL be capable of comparing two versions of a requirements graph and reporting differences between them.

##### Attributes:
- Parents: REQ-TRAQ-SYS-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-9 Requirement History

Reqtraq SHALL allow the user to view the list of changes made to each requirement.

##### Attributes:
- Parents: REQ-TRAQ-SYS-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-10 Requirement Deletion

Reqtraq SHALL allow requirements to be marked as deleted whilst retaining information within the requirements documents.

##### Attributes:
- Parents: REQ-TRAQ-SYS-2
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-11 Document Naming

Reqtraq SHALL enforce a naming scheme for requirements documents.

##### Attributes:
- Parents: REQ-TRAQ-SYS-3
- Rationale: Consistency helps in the development process to know where to find various certification documents.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-12 Requirement Numbering

Reqtraq SHALL enforce a numbering scheme for requirement identifiers.

##### Attributes:
- Parents: REQ-TRAQ-SYS-3
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-13 Schema Storage

Reqtraq SHALL allow a schema to be stored alongside the requirements and loaded.

##### Attributes:
- Parents: REQ-TRAQ-SYS-4
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-14 Schema Enforcement

Reqtraq SHALL allow the user to view reports showing where requirement documents do not meet the defined schema.

##### Attributes:
- Parents: REQ-TRAQ-SYS-4
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-15 Schema Definition

Reqtraq SHALL allow the structure of the requirements graph to be defined in the schema.

##### Attributes:
- Parents: REQ-TRAQ-SYS-4
- Rationale:
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-16 Command Line

Reqtraq SHALL have a command line interface.

##### Attributes:
- Parents:
- Rationale: Command line interface allows for integration with other development tools.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SWH-17 Web Interface

Reqtraq SHALL have a simple web interface that allows generation and filtering of reports.

##### Attributes:
- Parents:
- Rationale: Web interface allows for simple presentation of reports to auditors.
- Verification: Test
- Safety Impact: None
