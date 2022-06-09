# ReqTraq Test File

This file is used as a test input for the reqtraq tool.

## List Of Requirements

### REQ-TEST-SWH-1 Section 1

Body of requirement 1.

###### Attributes:
- Parents: REQ-TEST-SYS-1
- Rationale: Rationale 1
- Verification: Test 1
- Safety impact: Impact 1

### REQ-TEST-SWH-2 Section 2

Body of requirement 2.

###### Attributes:
- Parents: REQ-TEST-SYS-2
- Rationale: Rationale 2
- Verification: Test 2
- Safety impact: Impact 2

### ASM-TEST-SWH-1 An assumption

Assumptions have different attributes

###### Attributes:
- Parents: REQ-TEST-SWH-2
- Validation: Some validation strategy

### ASM-TEST-SWH-2 Another assumption

This one should fail because of an invalid parent

###### Attributes:
- Parents: REQ-TEST-SYS-2
- Validation: Some validation strategy

### ASM-TEST-SWH-3 One more assumption

This one should fail because of an invalid attribute

###### Attributes:
- Parents: REQ-TEST-SWH-2
- Verification: Test 2
