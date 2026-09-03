# The brief

From the product sponsor, verbatim:

> We're building **Framepost** — people follow photographers they like and
> see their new photos. Think simple: you post a photo with a caption, your
> followers see it in a feed, they can like it. Marketing projects **10
> million registered users by the end of year one**, mostly on phones.
> It has to feel instant and it can't be down during the launch campaign.
> Can you sketch how we'd build it?

That is all you get in writing. The sponsor is a busy human, not an oracle:
worksheet 1 collects the questions you would ask them; for everything else,
state an assumption and move on.

Facts the sponsor confirmed when pressed (use these; do not re-derive them):

- Photos only, no video. Originals average **~3 MB**; the app can display a
  resized version (**~300 KB**) in the feed.
- The audience is global, but year-one marketing targets **one region**.
- Deleting a photo must actually delete it (legal requirement in the target
  market).
- There is no budget position on build-vs-buy yet — assume you may use
  managed building blocks (object storage, managed databases, load
  balancers) as components in your sketch.
