# Design notes — taskd

Fill this in as you build. It is not decoration: the capstone review is a
conversation about these answers, and "it passed the tests" is not an answer
to any of them. Keep it short — a paragraph per heading, numbers where the
question asks for numbers.

## 1. Layer boundaries

Draw the dependency arrows between `cmd/taskd`, `httpapi`, `task`, and
`sqlitestore`. Which package would you have to touch to:

- add a `priority` field to a task?
- move storage from SQLite to Postgres?
- add a second transport (say, gRPC) beside HTTP?

If an answer is "more than one package", say which and why that is or is not
acceptable.

## 2. Error strategy

Where does an error first become an HTTP status code, and how many places in
the program know about status codes at all? Explain the rule you followed for
what reaches the client versus what reaches the log, and name one error whose
classification you had to think about.

## 3. Authentication and its threat model

What exactly does a bearer token in `TASKD_TOKENS` protect against, and what
does it not protect against? Cover at least:

- what an attacker who can read your traffic can do;
- what an attacker who can read your logs can do;
- how you would rotate a token without downtime;
- why the comparison is done the way it is.

## 4. A measured performance decision

Run the store benchmark, once as the schema ships and once with migration v2
(the index on `tasks(status)`) commented out — on a **fresh** database each
time, since migrations do not re-run:

```sh
go test -run '^$' -bench BenchmarkListByStatus -benchmem ./sqlitestore
```

| variant     | ns/op | B/op | allocs/op |
|-------------|-------|------|-----------|
| with index  |       |      |           |
| no index    |       |      |           |

What did you expect, what did you measure, and what would change your mind
about keeping the index? (Indexes are not free: name the cost.)

## 5. Observability in an incident

A client reports "the API is failing". Walk through what you would look at,
in order, using only what this service exposes: the access log, the metrics
endpoint, `/healthz`, `/readyz`. Which signal tells you *that* something is
wrong, which one tells you *where*, and which one lets you follow a single
failing request end to end?

## 6. Metric design

The registry in `httpapi/metrics.go` ships complete; what it exposes is your
call. Justify the three metrics you wired up:

- why `http_requests_total` carries `method`, `route` and `status` and nothing
  else — name one label you were tempted by and what it would have cost;
- why the `route` label is the mux pattern, with the number you would put on
  the alternative (how many series does `/tasks/{id}` become with a million
  tasks?);
- what question each end of `durationBuckets` answers, and which bound you
  would move first for this service;
- where `Instrument` sits relative to `Auth`, and what the dashboard shows
  during a credential rotation gone wrong if you nest them the other way.

## 7. Shutdown and the timeout budget

`Run` ships complete too. Trace a SIGTERM that arrives while a request is
mid-handler: who cancels what, what the client sees, and when the process
finally exits. Then justify the ordering
`requestTimeout < WriteTimeout < shutdownGrace` — including what breaks at
each pair if you swap it — and say what `Timeout` does *not* do to the
handler's goroutine.

## 8. What you would do next

Name the two changes you would make before putting this in front of real
users, and what stopped you from making them here.
