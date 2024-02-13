*** Settings ***
Resource                            resource_file.resc

*** Keywords ***
# A comment describing a keyword
A First Keyword
    Do Stuff
    Do More Stuff

A Second Keyword
    Do Even More Stuff

*** Test Cases ***

# @llr REQ-TEST-SWL-14
A Robot Test Case
    # Indentation matters
    A Second Keyword

# @llr REQ-TEST-SWL-14
Another Robot Test Case
    # Indentation matters
    A First Keyword
