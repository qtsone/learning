# An earlier review of PR #14 — rejected for tone

A colleague reviewed this PR before you. The maintainer withdrew the review
and asked for a redo: every technical observation below is *right*, and
every comment is unusable. Your job in part 2 is to rewrite each one so the
same point lands — same location, same severity — phrased about the code,
specific, and actionable.

---

**Comment 1** — on the `fmt.Sprintf` query in `Register`:

> Seriously? String-gluing SQL together after we JUST did the security
> training. Do you even read your own code before pushing, or do you just
> hope nobody looks?

**Comment 2** — on the `db.Exec(q)` line:

> You always do this. You've been told about ignoring errors at least twice
> now. I'm not approving anything of yours until you stop being sloppy.

**Comment 3** — top-level review summary:

> This whole PR is a mess. Rewrite it properly and maybe think a little
> before you open the next one.
