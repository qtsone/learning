In Go: option 2 is the tempting one-liner `q.items = q.items[1:]`. It reslices
in O(1), but the backing array is not freed while any slice still points into
it — dequeued elements stay reachable, so a busy queue quietly pins memory it
will never read again.
