**In Go:** both cost you one import — `net/http` + `encoding/json` on
one side, `protoc`-generated stubs + `google.golang.org/grpc` on the
other. Implementation effort is a wash. The real cost of the choice
lands on your *consumers*, which is exactly why audience decides.
