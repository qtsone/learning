In Go:

- macOS: `brew install go`, or download the `.pkg` installer from
  [go.dev/dl](https://go.dev/dl/) and click through it.
- Linux: prefer the official tarball (distro packages are often old). Follow
  the three steps at [go.dev/doc/install](https://go.dev/doc/install) — they
  unpack Go into `/usr/local/go` and add its `bin` directory to `PATH`.

Then verify:

```sh
go version
# go version go1.25.1 darwin/arm64
```
