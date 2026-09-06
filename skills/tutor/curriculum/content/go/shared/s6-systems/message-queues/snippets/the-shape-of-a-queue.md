In Go: a buffered channel looks like a queue and is not one. A value received
from a channel is *gone* — there is no ack, so a consumer that crashes after
`<-ch` silently loses the work, and there is no redelivery, no visibility
timeout, no dead-lettering. Channels move work between goroutines inside one
process; a message queue moves work between *processes that fail
independently*. The exercise makes you build the difference.
