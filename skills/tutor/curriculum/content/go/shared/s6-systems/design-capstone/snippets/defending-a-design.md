**In Go:** your API sketch does not need an implementation, but it does
need to be concrete enough to argue about — request and response
shapes, the idempotency key, the error cases. A handful of handler
signatures or a short `service`/`message` sketch is a fine level of
detail. Similarly, when your estimates need "what one node does",
prefer numbers you measured in S5 (your own benchmarks and profiles)
over the generic anchors from design-intro, and say which they are.
