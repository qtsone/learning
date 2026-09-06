In Go: ask Go where that second directory is with `go env GOPATH` — installed
tools land in the `bin` folder inside it (usually `~/go/bin`). The editor
lesson introduced language servers; Go's is called `gopls`, and when you
install the Go extension in VS Code it offers to install `gopls` — into
exactly that folder. So add this line to your shell profile now:

```sh
export PATH="$PATH:$HOME/go/bin"
```
