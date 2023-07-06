# ReqTraq Test File

This file is used as a test input for the reqtraq tool.

## Valid Requirements

### REQ-TEST-SYS-1 VALID System-level requirement, no parent

Shall be linked from component requirements in this document.

###### Attributes:
- Parents:
- Rationale: Rationale
- Component Allocation: System

### REQ-TEST-SYS-2 VALID System-level requirement, customer parent

Shall be linked from component requirements in this document.

###### Attributes:
- Parents: REQ-TEST-CST-1
- Rationale:
- Component Allocation: System

### REQ-TEST-SYS-3 VALID Component-level requirement, no parent

Component1. Shall be linked from high-level requirement.

###### Attributes:
- Parents:
- Rationale: Rationale
- Component Allocation: Component1

### REQ-TEST-SYS-4 VALID Component-level requirement, system parent

Component1. Shall be linked from high-level requirement.

###### Attributes:
- Parents: REQ-TEST-SYS-1
- Rationale:
- Component Allocation: Component1

### REQ-TEST-SYS-5 VALID Component-level requirement, customer parent

Component2. Shall be linked from high-level requirements.

###### Attributes:
- Parents: REQ-TEST-CST-1
- Rationale:
- Component Allocation: Component2

## Invalid Requirements

### REQ-TEST-SYS-6 INVALID System-level requirement, system parent

Invalid, links to a system-level requirement in this doc. And there is a SHALL mentioned in this body.

###### Attributes:
- Parents: REQ-TEST-SYS-1
- Rationale:
- Component Allocation: System

### REQ-TEST-SYS-7 INVALID Component-level requirement, component parent

Invalid, links to a component-level requirement in this doc. And there is a shall mentioned in this body.

###### Attributes:
- Parents: REQ-TEST-SYS-3
- Rationale:
- Component Allocation: Component1
