# Tutor notes — Local Dev with Compose

## Where the learner is

Third lesson of the containers pack. They can write a Dockerfile and a
multi-stage Go build; this is the first time two containers have to talk to
each other. From S4 they bring secrets hygiene — keep them out of source, read
them from the environment, fail loudly when one is missing — and SQL via
`database/sql`, but they have not written an HTTP server yet, so the provided
`app/` is scaffolding, not material: if they start reading `main.go` closely,
steer them back to `compose.yaml`.

The grade is static. `check.sh` never needs Docker, and its two live checks
report `skip` when no daemon is reachable. If they have Docker, push them to
run the stack by hand anyway — the `docker compose ps` moment where `db` goes
healthy *before* `app` starts is the lesson.

## Common misconceptions

- **"Services talk to each other through published ports."** They do not.
  `ports:` is host-to-container only; between services it is always the
  container port, and the app works even when `db` publishes nothing.
- **`localhost` inside a container.** Reaching for `localhost:5432` in
  `DATABASE_URL` is the standard first bug. Tie it back to lesson one: a
  container has its own network namespace, so `localhost` is itself.
- **"`depends_on` waits for the database."** It waits for the *container*.
  Ask what Postgres's first boot actually does (initdb, temporary server,
  restart) and why that loses the race on a fast machine.
- **"`service_healthy` means I don't need retries."** Startup ordering is not
  a runtime guarantee. The database can restart later; the client still needs
  backoff.
- **Bind-mounting the Postgres data directory** because "then I can see the
  files". Host uids, host filesystem semantics, and a database that will not
  start. Data goes in a named volume.
- **`down` vs `down -v`.** Many believe `down` wipes data; it does not, and
  that is why a stale volume can keep serving an old schema. `-v` is the
  "and my data too" flag.
- **`.env` vs `environment:`.** One is substituted into the file by Compose,
  the other is set inside the container. They are not the same variable space.
- **Bind mount over a scratch/distroless image.** Mounting source into an image
  with no toolchain does nothing — that is what `target: dev` is for.

## Grilling points

- "Your app dials `db:5432`. Walk me from that string to a TCP connection —
  who resolves `db`, and what happens to that answer when the container is
  recreated?"
- "Delete `ports:` from `db` entirely. Does the app still work? Why?"
- "You publish `"8080:8080"`. Your teammate already runs something on 8080.
  What breaks, and whose file has to change?"
- "Why is `pg_isready` a better healthcheck than `exit 0` — and what would a
  healthcheck for *your* Go service look like?"
- "You ran `down`, then `up`, and the database still had yesterday's rows.
  Where were they, and which command deletes them?"
- "Where does Compose stop being the right tool, concretely? Name two things
  it cannot do that you'd need in production." (Rescheduling on node failure,
  rolling updates/rollback, multi-host, reconciliation.)
- Stretch: "`$$POSTGRES_USER` in the healthcheck — why two dollar signs?"

## Grading rubric

- **A** — All 15 static checks pass; the compose file is tidy and the learner
  can explain, unprompted, why the db needs no published port, why data is a
  named volume while source is a bind mount, and what `service_healthy` does
  and does not promise. Bonus if they ran the stack and read `docker compose
  config` output.
- **B** — Checks pass, but explanation is mechanical in one area (typically
  volumes or the healthcheck) — they copied the shape without the reason.
  One round of grilling fixes it.
- **C** — Checks pass only after heavy hinting, or `localhost`/hard-coded
  password/naive `depends_on` appeared and was fixed by instruction rather
  than by reasoning. Pass only if a re-explanation lands.
- **Fail** — Checks failing, or the learner believes services talk over
  published ports, or cannot say what `down -v` destroys. Remediate: they will
  carry this straight into the Kubernetes lessons.

## Remediation ladder

1. "Read the FAIL line and its `fix:` aloud. Which of the twelve acceptance
   criteria is it about?"
2. "Where does that key belong — under a service, or at the top level?" (Most
   early failures are indentation: `volumes:` under `db` versus `volumes:` at
   the root are different things with the same name.)
3. For discovery: "In `DATABASE_URL`, what would that hostname resolve to
   *inside* the app container? Who else is listening there?" For ordering:
   "What does `depends_on` in list form actually wait for? Say it in terms of
   processes." For volumes: "Which of these two directories do you want to
   still exist tomorrow, and which do you edit in your editor?"
4. Give the shape of the one block they are stuck on — e.g. the `healthcheck`
   keys and what each means — but never the whole file; the file is the
   exercise.

## After passing

Preview: "One file, one machine, one `up`. When this pack resumes, you meet the
model that survives a machine dying — Kubernetes: pods, deployments, services,
and why nothing in it is a container command."
