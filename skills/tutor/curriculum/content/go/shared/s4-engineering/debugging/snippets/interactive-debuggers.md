In Go the debugger is **delve**. Install it once, then run your test suite
under it:

```sh
go install github.com/go-delve/delve/cmd/dlv@latest
cd exercise
dlv test                      # compile the tests, attach the debugger
```

A session looks like this — the commands to know:

```text
(dlv) break ledger.Lowest         # breakpoint at a function
(dlv) break ledger.go:24          #   …or at a file:line
(dlv) continue                    # run until a breakpoint hits
(dlv) next                        # step over: run this line, stop at the next
(dlv) step                        # step into the function being called
(dlv) print lowest                # show one variable
(dlv) locals                      # show every local in scope
(dlv) break ledger.go:30 if i == 8  # conditional breakpoint
(dlv) continue                    # …until the next hit; Ctrl-D or quit to leave
```

Your editor wraps the same engine: the VS Code Go extension from your S0
dev-environment lesson drives delve behind its "Debug Test" links, with
breakpoints as clickable margins. Learn the CLI once anyway — it works over
ssh, in containers, everywhere the mouse can't reach.
