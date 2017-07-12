# Reqtraq


Reqtraq is an open source go tool for requirement management, as mandated by
DO-178C.
Reqtraq is designed to stay out of your way. It requires no user interaction for day-to-day tasks.
Instead, it parses documents that are required in the certification process and extracts everything
it needs from there.

Reqtraq has 3 main components:
1. **Precommit hook** makes sure the documents have the correct structure and linking:
   * Files correctly named
   * Requirements correctly formatted and continuous
   * References exist, Parent requirements exist and not DELETED
   * Required attributes are there and correctly formatted

2. **Prepush hook** exports tasks to desired task management tool (currently supports Phabricator; JIRA and others need to be added)

3. **Standalone binary**
   * Report generation with filtering
   * Phabricator export
   * Web tool for easy inspection



## How to install Reqtraq
### Dependencies
  * go 1.8+ *Installation instructions [here](https://golang.org/doc/install).*
  * pandoc *Installation instructions [here](https://pandoc.org/installing.html).*


### Installation
```
$ go get github.com/daedaleanai/reqtraq
$ export PATH=$PATH:$GOPATH/bin
```

## Using Reqtraq
Reqtraq is tightly integrated with Git. See the certification documents in the `certdocs` directory for some good examples.
Reqtraq uses the Git history to figure out the Git commits associated with a requirement and the Phabricator API to assess the completion status of each requirement.

### Usage examples
#### Getting the next available requirement ID
```
$ reqtraq nextid certdocs/TRAQ-138-SDD.md
REQ-TRAQ-SWL-21
```

#### Parse and List requirements
```
$ reqtraq list certdocs/TRAQ-100-ORD.md
Requirement REQ-TRAQ-SYS-1  Bidirectional tracing.
...
```

#### Report generation
In report tags such as 'Changelists' and 'Problem Reports' will not work if not integrated with a task manager such as Phrabricator etc. (currently supported for Phabricator; JIRA and others need to be added)
```
$ reqtraq reportdown
2017/06/06 22:48:12 Creating ./req-down.html (this may take a while)...
...
```
Filtering:
```
$ reqtraq reportdown --id_filter=".*TRAQ-SYS.*"
2017/06/06 22:51:23 Creating ./req-down.html (this may take a while)...
2017/06/06 22:51:41 Creating ./req-down-filtered.html (this may take a while)...
```

#### Start the web interface
```
$ reqtraq web :8080
Server started on http://localhost:8080
```

## Getting help
```
$ reqtraq help
```
Shows general help and a list of commands
```
$ reqtraq help <command>
```
Displays help on a specific command.
