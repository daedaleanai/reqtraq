# Overall Requirements Document for Reqtraq

## Introduction

Reqtraq is a <abbr title="Requirements Management Tool">RMT</abbr>.

### Purpose

The purpose of this document is to define the system requirements for the Reqtraq tool. The system requirements will define the high-level needs for a requirements management tool in order to be used within a aircraft certification project and will be used for development of the software requirements documents.

### Scope

This document defines at the highest level the requirements for a requirements management tool on a certifiable aircraft system project.

### Applicable Documents

#### External Documents

N/A

#### Internal Documents

**DDLN-1-DS** Daedalean Documentation Standards

**DDLN-6-RS** Daedalean Requirements Standards

### Nomenclature and Description of Terms

#### Description of Terms

N/A

## Reqtraq System Overview

### System Goals

Reqtraq's goal is to ensure bidirectional traceability throughout all levels of requirements and to identify faults in that traceability. In fulfilling these goals, Reqtraq generates various reports and prevents the introduction of errors by requiring compliance with the Reqtraq system.

## System Requirements Identification

### Functional Requirements

#### REQ-TRAQ-SYS-1 Traceability

Reqtraq SHALL allow "requirement to implementation" and "requirement to verification" traceability to be recorded, reviewed and visualized.

##### Attributes:
- Rationale: The status of implementation and verification against the requirements must be demonstrated for certification. Easy access to information during development is vital to prevent problems being missed.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SYS-2 Change History

Reqtraq SHALL allow the change history of requirements to be recorded, reviewed and visualized.

##### Attributes:
- Rationale: Easy access to the evolution of requirements allows for the cause of the introduction of faults to be identified.
- Verification: Test
- Safety Impact: None

#### REQ-TRAQ-SYS-3 Unique Identifiers

Reqtraq SHALL ensure that requirement identifiers are consistent and are not repeated or reused at any level.

##### Attributes:
- Rationale: Strict enforcement of identifiers helps prevent traceability issues.
- Verification: Test
- Safety Impact: None


#### REQ-TRAQ-SYS-4 Configurable

Reqtraq SHALL allow rules on the format and relationship of requirement documents to be defined and enforced.

##### Attributes:
- Rationale: The exact layout of requirements depends on the project structure, the layout should be defined and enforced.
- Verification: Test
- Safety Impact: None
