# Reqtraq


Reqtraq is an open source Go tool for requirement management.
Reqtraq is designed to stay out of your way. It requires no user interaction for day-to-day tasks.
Instead, it parses documents that are required in the certification process and extracts everything
it needs from there.

Reqtraq has two main use-cases:
1. **Git hook** makes sure the documents have the correct structure and linking:
   * Files correctly named
   * Requirements correctly formatted and continuous
   * References exist, Parent requirements exist and not DELETED
   * Required attributes are there and correctly formatted

2. **Standalone binary**
   * Report generation with filtering
   * Web tool for easy inspection



## How to install Reqtraq
### Dependencies
  * [go 1.17+](https://golang.org/doc/install)
  * [pandoc](https://pandoc.org/installing.html)
  * [universal-ctags](https://github.com/universal-ctags/ctags/blob/master/README.md#the-latest-build-and-package) *Note there is also the unmaintained exuberant-ctags which should be avoided.*
  * [clang+llvm-14](https://github.com/llvm/llvm-project/releases/tag/llvmorg-14.0.0), *only needed for building reqtraq with support for libclang.*

### Installation
Basic installation can be done with:
```
$ go install github.com/daedaleanai/reqtraq@latest 
$ export PATH=$PATH:$GOPATH/bin
```

For repos having requirements documents defined in `reqtraq_config.json` with `"codeParser": "clang"` reqtraq needs libclang support. Install the `libclang-14-dev` package on Ubuntu, or download clang+llvm-14 from their release page and unpack it somewhere. Set `LLVM_LIB` accordingly and build reqtraq enabling the "clang" tag:
```
$ LLVM_LIB=/usr/lib/llvm-14/lib
$ LLVM_LIB=~/clang+llvm-14.0.0-x86_64-linux-gnu-ubuntu-18.04/lib
$ export CGO_LDFLAGS="-L${LLVM_LIB} -Wl,-rpath=${LLVM_LIB}"
$ go install --tags clang github.com/daedaleanai/reqtraq@latest 
$ export PATH=$PATH:$GOPATH/bin
```

## Using Reqtraq
Reqtraq is tightly integrated with Git. See the certification documents in the `certdocs` directory for some good examples.
Reqtraq uses the Git history to figure out the Git commits associated with a requirement and the Phabricator API to assess the completion status of each requirement.

### Usage examples
#### Getting the next available requirement ID
```
$ reqtraq nextid certdocs/TEST-138-SDD.md
REQ-TEST-SWL-21
```

#### Parse and List requirements
```
$ reqtraq list certdocs/TEST-100-ORD.md
Requirement REQ-TEST-SYS-1  Bidirectional tracing.
...
```

#### Report generation
In report tags such as 'Changelists' and 'Problem Reports' will not work if not integrated with a task manager such as Phrabricator etc. (currently supported for Phabricator; JIRA and others need to be added)
```
$ reqtraq report down
2017/06/06 22:48:12 Creating ./req-down.html (this may take a while)...
...
```
Filtering:
```
$ reqtraq report down --id_filter=".*TEST-SYS.*"
2017/06/06 22:51:23 Creating ./req-down.html (this may take a while)...
2017/06/06 22:51:41 Creating ./req-down-filtered.html (this may take a while)...
```

#### Start the web interface
```
$ reqtraq web :8080
Server started on http://localhost:8080
```

#### Configuration
Reqtraq is configured using a `reqtraq_config.json` file in the root of the repository that contains both requirements and data.

```json
{
    "repoName": "reqtraq",
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
            "path": "certdocs/TRAQ-100-ORD.md",
            "prefix": "TRAQ",
            "level": "SYS",
        },
        {
            "path": "certdocs/TRAQ-137-SRD.md",
            "prefix": "TRAQ",
            "level": "SWH",
            "parent": {
                "prefix": "TRAQ",
                "level": "SYS"
            }
        },
        {
            "path": "certdocs/TRAQ-138-SDD.md",
            "prefix": "TRAQ",
            "level": "SWL",
            "parent": {
                "prefix": "TRAQ",
                "level": "SWH"
            },
            "implementation": {
                "archs": {
                    "armv6m": {
                        "compilationDatabase": "",
                        "compilerArguments": ["--target=thumbv6m-linux-eabi"]
                    },
                    "linux-x64": {
                        "compilationDatabase": "/BUILD/OUTPUT/compile_commands.json",
                        "compilerArguments": []
                    }
                },
                "code": {
                    "paths": ["."],
                    "matchingPattern": ".*\\.go$",
                    "ignoredPatterns": [".*_test\\.go$"],
                    "archPatterns": {
                        "armv6m": {
                            "paths": ["src", "include"],
                            "matchingPattern": ".*/armv6m/.*((.*\\.cc$)|(.*\\.c$)|(.*\\.hh$))",
                            "ignoredPatterns": [".*_test\\.cc$"]
                        },
                        "linux-x64": {
                            "paths": ["src", "include"],
                            "matchingPattern": ".*/linux-x64/.*((.*\\.cc$)|(.*\\.c$)|(.*\\.hh$))",
                            "ignoredPatterns": [".*_test\\.cc$"]
                        }
                    }
                },
                "tests": {
                    "paths": ["."],
                    "matchingPattern": ".*_test\\.go$",
                     "archPatterns": {
                        "armv6m": {
                            "paths": ["src"],
                            "matchingPattern": ".*/armv6m/.*_test\\.cc$"
                        },
                        "linux-x64": {
                            "paths": ["src"],
                            "matchingPattern": ".*/linux-x64/.*_test\\.cc$"
                        }
                    }
                }
            }
        }
    ]
}
```

All document paths are specified with respect to the root of the repository they belong to. It is possible 
to separate code and requirements across multiple repositories with reqtraq by specifying parent 
and children repositories in its configuration file. 

There must always be a top level repository that contains a configuration file with no parents.

Child repository configuration:
```json
{
    "repoName": "childRepo",
    "parentRepository": {
        "repoName": "parentRepo",
        "repoUrl": "/path/to/parent/repo"
    },
    "documents": [
        {
            "path": "certdocs/TEST-138-SDD.md",
            "prefix": "TRAQ",
            "level": "SWL",
            "parent": {
                "prefix": "TRAQ",
                "level": "SWH"
            },
            "attributes": [
                {
                    "name": "Parents",
                    "required": "any",
                    "value": "REQ-TEST-SWH-(\\d+)"
                }
            ],
            "implementation": {
                "archs": {
                    "armv6m": {
                        "compilationDatabase": "",
                        "compilerArguments": ["--target=thumbv6m-linux-eabi"]
                    },
                    "linux-x64": {
                        "compilationDatabase": "/BUILD/OUTPUT/compile_commands.json",
                        "compilerArguments": []
                    }
                },
                "code": {
                    "paths": ["code"],
                    "matchingPattern": ".*\\.(cc|c|h|hh)$",
                    "ignoredPatterns": [".*_test\\.(cc|c|h|hh)$"],
                    "archPatterns": {
                        "armv6m": {
                            "paths": ["src", "include"],
                            "matchingPattern": ".*/armv6m/.*((.*\\.cc$)|(.*\\.c$)|(.*\\.hh$))",
                            "ignoredPatterns": [".*_test\\.cc$"]
                        },
                        "linux-x64": {
                            "paths": ["src", "include"],
                            "matchingPattern": ".*/linux-x64/.*((.*\\.cc$)|(.*\\.c$)|(.*\\.hh$))",
                            "ignoredPatterns": [".*_test\\.cc$"]
                        }
                    }
                },
                "tests": {
                    "paths": ["."],
                    "matchingPattern": ".*_test\\.(cc|c|h|hh)$",
                    "ignoredPatterns": [],
                     "archPatterns": {
                        "armv6m": {
                            "paths": ["src"],
                            "matchingPattern": ".*/armv6m/.*_test\\.cc$"
                        },
                        "linux-x64": {
                            "paths": ["src"],
                            "matchingPattern": ".*/linux-x64/.*_test\\.cc$"
                        }
                    }
                }
            }
        }
    ]
}
```

Parent repository configuration:
```json
{
    "repoName": "parentRepo",
    "childrenRepositories": [
        {
            "repoName": "childRepo",
            "repoUrl": "/path/to/child/repo"
        },
        {
            "repoName": "anotherChildRepo",
            "repoUrl": "/path/to/another/child/repo"
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
            "path": "certdocs/TEST-100-ORD.md",
            "prefix": "TRAQ",
            "level": "SYS",
            "attributes": []
        },
        {
            "path": "certdocs/TEST-137-SRD.md",
            "prefix": "TRAQ",
            "level": "SWH",
            "parent": {
                "prefix": "TRAQ",
                "level": "SYS"
            }
        }
    ]
}
```

Accepted references to parent and child repositories are:
- A file system path which contains a git checkout.
- A URL to a git repository.

## Getting help
```
$ reqtraq help
```
Shows general help and a list of commands
```
$ reqtraq help <command>
```
Displays help on a specific command.
