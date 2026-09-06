**In Go:** nothing about this choice is language-specific — that is
the point — but you have lived one side of it: the `pgx` transaction
in your S5 service that wrote a charge and its idempotency key in
one commit is a multi-row invariant. Before moving any dataset to a
key-value API, find every transaction in your code that spans it and
something else: each one is a consistency requirement you would be
giving up.
