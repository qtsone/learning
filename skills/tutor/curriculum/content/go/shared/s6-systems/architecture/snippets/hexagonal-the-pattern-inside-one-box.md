**In Go:** consumer-defined interfaces make the dependency rule nearly
free — the domain package declares the small interfaces it needs, and
adapters implement them without being imported:

```text
orders/           domain: Order, PlaceOrder; ports: Store, Payments, Events
orders/postgres/  adapter: implements orders.Store using pgx
orders/httpapi/   adapter: handlers mapping JSON onto orders calls
main.go           wires concrete adapters into the domain at startup
```

`orders` imports neither `net/http` nor `pgx` — check its import list
to *verify* the dependency rule. Its tests use an in-memory `Store`
and a fake clock: the same shape as your S5 service tests.
