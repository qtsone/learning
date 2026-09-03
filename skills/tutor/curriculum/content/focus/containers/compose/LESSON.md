# Local Dev with Compose

> `focus.containers.compose` · ~1.5-2.5h · Stage: Focus: Containers I

## Objectives

By the end of this lesson you can:

- Implement a compose file that runs a Go app together with a database and
  wires them via service names on a shared network.
- Explain how Compose DNS-based service discovery lets the app reach the
  database by service name instead of an IP.
- Choose between named volumes and bind mounts for database persistence and
  live code reloading, and justify each choice.
- Use `depends_on` with a healthcheck so the app starts only after the database
  is ready, and explain why `depends_on` alone is insufficient.
- Manage the stack lifecycle with `up`/`down`/`logs`/`exec`, including tearing
  down with and without volumes.

## The command line that got too long

You can already build an image and run it. But the moment your service needs a
database, `docker run` turns into a paragraph — a network to create first, an
`--env` per setting, a `-p`, a `-v`, a `--name` so the other container can find
it, then a second paragraph for the database. You write it down in a README,
the README rots, and a new teammate loses an afternoon.

**Compose** is that paragraph, in a file, under version control, and
`docker compose up` is then the entire setup document for the project. The file
is `compose.yaml` (the old name `docker-compose.yml` still works) and the tool
is the `docker compose` subcommand — the separate `docker-compose` binary was
v1 and is retired.

Be clear about what Compose is for, because the industry is not: it runs
containers **on one machine**. Excellent for development, for integration tests
in CI, and for a small single-box deployment. Not a production orchestrator —
no rescheduling when a machine dies, no rolling update you would bet a release
on, no controller reconciling desired state. That is Kubernetes, later in this
pack.

## Anatomy

```yaml
services:            # the containers, keyed by name
  app: { }
  db: { }
volumes:             # named storage Docker manages for you
  dbdata:
networks:            # usually omitted: you get a default one
```

Each key under `services` is a service name, doing more work than you think: it
names the container, it names the DNS record, and it is what you type in
`docker compose logs <name>`. The whole file is one **project**, named after
the directory (or by `name:` at the top). Every object Compose creates is
prefixed with the project name, which is why two projects on one machine do not
collide.

## The default network: names, not IPs

Compose creates one network for the project and attaches every service to it.
On that network, **the service name resolves to the container's IP** through
Docker's embedded DNS. Your app connects to `db:5432` and that is the whole
service-discovery story: no IP anywhere, no config to update when the container
is recreated with a different address.

Two consequences worth burning in:

- **`localhost` inside a container is that container.** A container is a
  process with its own network namespace (lesson one of this pack). If your app
  dials `localhost:5432` it is dialing itself, and you get "connection
  refused" — the single most common first Compose bug.
- **Container-to-container traffic uses the container port**, always, whether
  or not you published anything. `ports:` is only about the host. The app
  reaches `db:5432` even when the database publishes nothing at all.

## Publishing without fighting your machine

```yaml
ports:
  - "${APP_PORT:-8080}:8080"   # host : container
```

The left side is a port on *your* machine, and it is a global resource: if 8080
is taken, `up` fails. A hard-coded host port in a committed file is a collision
waiting for the second person on the team; a variable with a default keeps the
file working out of the box and lets anyone override it in their own `.env`.

The database usually needs **no published port at all** — the app reaches it
over the compose network. Publishing `5432:5432` is how you discover that some
other project's Postgres has been running on your machine all along. If you
want `psql` from the host, publish `"${DB_PORT:-55432}:5432"` instead.

## Environment: two things with similar names

1. **`environment:` / `env_file:`** set variables *inside the container*. This
   is your app's configuration — the same "read it from the environment, never
   from source" habit S4's security lesson gave you, now written down in a file
   instead of typed at a prompt.
2. **A `.env` file next to `compose.yaml`** is read by Compose itself and
   substituted into the file *before* it is parsed:

```yaml
environment:
  DATABASE_URL: "postgres://app:${POSTGRES_PASSWORD}@db:5432/appdb?sslmode=disable"
  POSTGRES_PASSWORD: "${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD in .env}"
```

`${VAR}` substitutes, `${VAR:-default}` falls back, and `${VAR:?message}` fails
loudly when the variable is missing — fail-fast configuration, the same
instinct as validating config at startup. When you need a literal dollar sign
to survive as far as the container's own shell, double it: `$$POSTGRES_USER`.

This split is what keeps secrets out of git. `compose.yaml` is committed and
holds only `${...}` placeholders; `.env` holds the values and is git-ignored;
`.env.example` is committed as documentation of which keys must exist. (Add
`.env` to your `.gitignore` before you write a password into it, not after —
S4's security lesson on what a repository remembers forever applies here.)

## Volumes: named, bind, anonymous

A container's own filesystem dies with the container. Anything that must
outlive it goes in a volume, and there are three kinds:

| Kind | Written as | Lives where | Use for |
|---|---|---|---|
| Named | `dbdata:/var/lib/postgresql/data` | Docker-managed area, `docker volume ls` | data |
| Bind | `./app:/src` | an exact path on your host | source, config |
| Anonymous | `/var/lib/postgresql/data` | Docker-managed, unnamed | nothing, on purpose |

The rule of thumb is **bind mounts for source, named volumes for data**, and
the reasons are concrete. A bind mount is a hole punched from the host
filesystem into the container, so it carries host uids, host permissions, and
on macOS and Windows a translation layer that is both slower and subtly
different from Linux. That is fine for source you edit in your editor and
irritating-to-catastrophic for a database's data directory: the classic bug is
Postgres refusing to start because a host-mounted directory has the wrong
ownership, or a database file corrupted by a filesystem that does not honour
`fsync` the way it promises.

A named volume avoids all of it: Docker owns the storage, the container writes
to a normal Linux filesystem, and the data survives `docker compose down` —
which stops and removes containers but keeps named volumes. Only `down -v`
deletes them; learn that flag as "and my data too". (Anonymous volumes — a
mount path with no source — work, but leave unnamed storage behind that nobody
ever cleans up. Name them.)

One catch for a compiled language: bind-mounting your source into an image that
contains only a compiled binary achieves nothing — there is no compiler in
there to use it. That is why the provided `Dockerfile` has a `dev` stage with
the toolchain, and why your app service selects it with `target: dev`.

## `depends_on`: started is not ready

```yaml
depends_on:
  - db          # naive: waits for the container to start, nothing more
```

This form waits until the db *container* is running — for Postgres, roughly
200 ms before it accepts connections, and the first boot is worse, because the
official image runs `initdb`, starts a temporary server for initialisation,
shuts it down, and only then starts the real one. Your app connects into that
gap and dies. It "works on my machine" precisely when the machine is slow
enough to lose the race.

The fix is to define what *ready* means, and wait for that:

```yaml
db:
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U $$POSTGRES_USER -d $$POSTGRES_DB"]
    interval: 5s
    timeout: 3s
    retries: 5
    start_period: 10s
app:
  depends_on:
    db:
      condition: service_healthy
```

A healthcheck is a command Docker runs *inside* the container on a schedule;
exit code 0 means healthy. `interval` is how often, `retries` is how many
consecutive failures flip it to unhealthy, and `start_period` is a grace window
during which failures do not count against those retries — exactly right for a
slow first boot. The long form of `depends_on` then takes a `condition`:
`service_started` (the naive behaviour), `service_healthy`, or
`service_completed_successfully` for one-shot jobs like migrations.

Be honest about what this buys you: **startup ordering, not a dependency
guarantee.** The database can still restart at 3pm on a Tuesday while your app
is running. Ordering at boot is a convenience; retry-with-backoff in the client
is the actual correctness mechanism, and you will still need it.

## Lifecycle vocabulary

```sh
docker compose up -d              # start everything, detached
docker compose up -d --build      # rebuild images first
docker compose up -d --wait       # ...and block until healthchecks pass
docker compose ps                 # what is running, and its health
docker compose logs -f app        # follow one service's logs
docker compose exec db psql -U app appdb    # a shell in a running container
docker compose run --rm app go test ./...   # a one-off container, then discard
docker compose down               # stop and remove containers and the network
docker compose down -v            # ...and delete the named volumes (data gone)
docker compose config             # print the merged, interpolated file
```

`docker compose config` is the debugging tool most people discover far too
late: it shows you the file *after* `.env` substitution and defaults, which
answers "is that variable actually set?" in one second.

## Exercise

Open [`exercise/`](exercise/). You are given:

- `app/` — a tiny HTTP service (`/healthz`, and `/dbping`, which opens a TCP
  connection to the host in `DATABASE_URL` and reports what happened) plus a
  three-stage `Dockerfile` whose middle stage, `dev`, keeps the toolchain and
  runs `go run .`.
- `.env.example` — the keys your `.env` must define.
- `compose.yaml` — a starter with two services and a lot of TODOs.
- `check.sh` — the referee.

Write `compose.yaml` (and `.env`) so that:

1. The file is valid YAML with two services, `app` and `db`.
2. `app` builds from context `./app` with `target: dev`.
3. `db` runs a **pinned** Postgres image — a real version tag or a digest,
   never `latest` and never untagged.
4. `app` gets a `DATABASE_URL` whose host is the **service name** `db`, not
   `localhost` and not an IP.
5. No password is hard-coded in `compose.yaml`: interpolate
   `${POSTGRES_PASSWORD}` (or use `env_file:`), and give `db` the
   `POSTGRES_*` variables the image needs.
6. `.env` exists (copy `.env.example`), defines every variable you interpolate
   without a default, and does not still say `change-me`.
7. `db` has a healthcheck using `pg_isready`, with at least `interval` and
   `retries`.
8. `app` declares `depends_on: db: condition: service_healthy` — the long form.
9. Postgres data lives on a **named** volume mounted at
   `/var/lib/postgresql/data`, declared under the top-level `volumes:` key.
10. `app` **bind-mounts** `./app` into the container (`./app:/src`) so your
    edits are what the dev stage compiles.
11. `app` publishes a host port whose host side is a variable
    (`"${APP_PORT:-8080}:8080"`), container side 8080.
12. `db` does not claim the host's 5432.

**How the referee reads your YAML.** PyYAML is not guaranteed on your machine,
so `check.sh` carries a small reader covering the subset a compose file needs:
**spaces only, never a tab; two spaces per level; a space after every colon and
after every `- `; no anchors (`&`), aliases (`*`) or merge keys (`<<:`)**. Flow
style (`{cpu: 100m}`, `[a, b]`) is fine for short values, and a block sequence
may sit at its key's own indentation or one level in — both are legal YAML and
both are read. If the reader ever rejects legal YAML, that is an exercise bug,
not yours — tell your tutor.

Then run the referee, from inside `exercise/`:

```sh
bash check.sh
```

It grades the files you wrote by parsing them — **no Docker daemon, no network
and no root required** — so it works everywhere and it is the grade. Expect it
to fail on the starter; that is the point.

You **should** install Docker anyway — a container lesson you never run is a
container lesson you half-learned. With a daemon reachable, `check.sh` adds two
bonus live checks: `docker compose config`, and a real `up --wait` + `/dbping`
smoke test (which tears the stack down with `-v` afterwards, so keep nothing
precious in that volume). Without one, both are reported `skip`, never `FAIL`,
and the summary line says how many were skipped for missing tooling.

With Docker present, also do this by hand — it is where the understanding
lands:

```sh
docker compose up -d --build
docker compose ps                       # watch db go healthy before app starts
docker compose logs -f app
docker compose exec app wget -qO- localhost:8080/dbping
docker compose down                     # then up again: your data is still there
docker compose down -v                  # then up again: it is not
```

## Further reading

- [The Compose Specification](https://github.com/compose-spec/compose-spec/blob/main/spec.md)
- [Compose file reference — services](https://docs.docker.com/reference/compose-file/services/)
- [Control startup order](https://docs.docker.com/compose/how-tos/startup-order/)
- [Volumes](https://docs.docker.com/engine/storage/volumes/)
