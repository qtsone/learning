**In Go:** replication makes read routing an *application* concern.
Your S5 service held one `pgxpool` for one DSN; with replicas there
are several pools, and "may this query read stale data?" becomes a
parameter of your storage layer's API — the refund status read after
a support action goes to the leader; the nightly reconciliation walk
is happy on a lagging replica. Design the seam now or grep for every
query later.
