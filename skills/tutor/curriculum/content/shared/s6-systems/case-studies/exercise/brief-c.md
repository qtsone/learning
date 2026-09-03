# Brief C — Threadline (speed round)

*Sealed. Do not read this file until your 30-minute timer is running — the
whole point of Case C is producing five phases from a cold prompt.*

Threadline is a social app. Users follow accounts; opening the app shows a
feed of recent posts from the accounts they follow, newest first.

### Confirmed facts

- **40,000,000 daily active users**; median user follows 200 accounts.
- A few thousand accounts have **more than 5,000,000 followers**; the
  largest has 25,000,000.
- **8,000,000 posts/day**. A post is ~300 bytes of text plus zero or one
  image (~200 KB after resizing).
- Users open the feed ~6 times/day and read the top ~20 posts each time.
- Feed must render its first 20 posts in **under 1 second at p95**; a post
  may take up to a minute to reach followers.

Everything else is yours to assume. Thirty minutes, five phases, go.
