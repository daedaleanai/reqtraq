## Development

* After installing `reqtraq` with `go install github.com/daedaleanai/reqtraq`, create a branch in `$GOPATH/src/github.com/daedaleanai/reqtraq` and make changes.
* Test your changes with `go test`
* Add unit tests for any new features added
* Push to a forked repository then make a pull request from there to `daedaleanai/reqtraq`


## Code Style

* Source code must comply with `gofmt`'s code style. Run `gofmt -l -e .` to automatically correct any formatting errors before commiting code to reqtraq.
* Fail as fast as possible. For example, this is preferred

```
if not ok {return error}
```

instead of

```
if ok {
 ...
} else {
    return error
}
```
