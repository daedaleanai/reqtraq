# Overall Requirements Document for Reqtraq

Document Approval:
- Engineering, Program Manager: Luuk van Dijk
- Engineering, Engineer: Daniel Danciu
- Quality, Quality Engineer: Anna Chernova

## Introduction

Reqtraq is our <abbr title="Requirements Management Tool">RMT</abbr>.

### Purpose

The purpose of this document is to define the requirements for the Reqtraq
system. The requirements will define the system needs and will be used for
development of the hardware and software requirements documents. It complies
with section 5.3 of SAE-4754A / ED-79A and Daedalean AG’s <abbr title="Software
Requirements Standards">SRS</abbr>.

The purpose of the Reqtraq tool is to comply with the DO-178C / ED-12C
traceability data requirements.

### Scope

This document discusses the following topics:
- Report generation
- Traceability of requirements data, change data
- Compliance with formatting
- Assumptions

### Applicable Documents

#### External Documents

**FAA Order 8110.49** Software Approval Guidelines. 2003.

**RTCA DO-178C / EUROCAE ED-12C** Software Considerations in Air-borne Systems and Equipment Certification.

**IEEE 830-1998 Appendix A** Recommended Practice for Software Requirements Specifications. IEEE Standards Association. 2009.

**Developing Safety-Critical Software: A Practical Guide for Aviation Software and DO-178C Compliance.** Rierson, Leanna. 2013

**EUROCAE ED-79A / SAE ARP 4754A** Guidelines for Development of Civil Aircraft and Systems. 2010.

#### Internal Documents

**DDLN-6-SRS** Software Requirements Standards

**DDLN-1-DS** Documentation Standards

### Nomenclature and Description of Terms

#### Description of Terms

Changelist
  : a git commit resulting in the creation, revision, or deletion of any version controlled material.

Problem report
  : a change request identifying a missing, non-functioning, or non-compliant feature.

## Reqtraq System Overview

### System Goals

Reqtraq's goal is to ensure bidirectional traceability throughout all levels of
requirements and to identify faults in that traceability. In fulfilling these
goals, Reqtraq generates various reports and prevents the introduction of errors
by requiring compliance with the Reqtraq system.

## Requirements Overview

### Functional Requirements

Functional requirements identify what is necessary to obtain the desired
performance of the system under the listed operational modes and conditions.
This category includes and is developed through a combination of customer,
operational, performance, physical and installation, maintainability, security,
and interface restrictions, wishes, and requirements. Functional requirements
are defined and written according the section 5.3 of ARP 4754A / ED-79A.


##### REQ-TRAQ-SYS-7 Requirement document types

The RMT SHALL allow tracking of requirements within the following document types:

- Overall Requirements Document, which contains <abbr title="System-Level Requirements">SYS</abbr>
- Software/Hardware Requirements Documents, which contain <abbr title="High-Level Requirements">HLR</abbr>
- Software/Hardware Design Documents, which contain <abbr title="Low-Level Requirements">LLR</abbr>.

###### Attributes:
- Rationale: These documents contain the three levels of requirements, at varying levels of product development, that Reqtraq parses to establish traceability.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-8 Requirement document hierarchy

The hierarchical relationship of requirements documents SHALL be as follows:

- Software/Hardware Design Documents will have exactly one parent Software/Hardware Requirements Document.
- Software/Hardware Requirements Documents will have exactly one parent Overall Requirements Document.
- Overall Requirements Documents will have zero or one parent Overall Requirements Document.

###### Attributes:
- Rationale: This structure ensures traceability from high-level concepts to implementation.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-1 Bidirectional tracing

The RMT SHALL be able to generate reports of requirements ordered from system to
high level to low level requirement, then further down to implementation and to
test cases, and in reverse.

Further down is not needed because there is a 1-1 relationship between test
cases, test procedures, and verification reports.

###### Attributes:
- Rationale: The certification authority must be able to trace the requirements of different types and code to the others they are connected to.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-10 Implementation status

The RMT SHALL be able to show the implementation status for a requirement.

###### Attributes:
- Rationale: Bidirectional traceability is then equivalent to completeness of the paths.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-2 DELETED


##### REQ-TRAQ-SYS-3 Change history

The RMT SHALL be able to report the complete history of the changes for a given
set of requirements.

###### Attributes:
- Rationale: Certification authority must be able to verify that each requirement was appropriately implemented.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-4 DELETED


##### REQ-TRAQ-SYS-5 DERIVED

This is the placeholder System Requirement for derived HLRs.

###### Attributes:
- Rationale: Derived HLRs must have a parent requirement.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-6 Requirement attributes

The RMT SHALL enforce rules for the attributes of requirements.

###### Attributes:
- Rationale: Some attributes are optional and others are mandatory.
- Verification: Test.
- Safety impact: None.
- Type: SW


##### REQ-TRAQ-SYS-9 DELETED


### Other Assumptions

In the creation of these requirements it was assumed that Reqtraq users use Git for version control.
