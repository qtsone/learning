**In Go:** the standard library ships a doubly linked list
(`container/list`), yet you'll rarely see it in production Go — slices win
almost every benchmark that doesn't specifically need O(1) splicing. You are
building a list from scratch here not because Go needs another one, but
because pointer-linked nodes are the atom that stacks, queues, trees, and
graphs — the rest of this stage — are built from.
