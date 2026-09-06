# Documentation notes — fill every section, then bring this to the discussion

## Part 1 — Stranger test

Questions a competent stranger cannot answer from the old README (at least
six):

1.
2.
3.
4.
5.
6.

The one useful, true sentence in the old README worth keeping, and where it
went in my rewrite:

## Part 2 — Doc comments

| identifier | what the old comment failed to say (the contract points my version adds) |
|------------|---------------------------------------------------------------------------|
| package store |  |
| ErrNotFound   |  |
| Store         |  |
| Open          |  |
| Add           |  |
| Get           |  |
| List          |  |

A sentence I drafted and then cut because no test could falsify it:

## Part 3 — ADR

(The ADR itself lives in `snippets/docs/adr/0001-single-json-file-storage.md`.)

What would have to change about this project for ADR 0001 to deserve a
superseding ADR:

## Part 4 — Changelog curation

| commit | category (Added / Changed / Fixed / omit) | breaking? for whom? | why |
|--------|-------------------------------------------|---------------------|-----|
| 9f3c2a1 |  |  |  |
| 41d0b77 |  |  |  |
| b2e91c4 |  |  |  |
| 7ac00d3 |  |  |  |
| c31f9e0 |  |  |  |
| 88aa412 |  |  |  |
| 5b7ca29 |  |  |  |
| e0d51f2 |  |  |  |

(The entry itself goes in `snippets/CHANGELOG.md` under `[0.2.0]`.)

## Part 5 — Placement drill

| information | home (README / doc comment / ADR / changelog / none) | why — one sentence |
|-------------|-------------------------------------------------------|--------------------|
| The exact commands to install the tool on a fresh machine |  |  |
| What `Get` returns when no snippet has the given name |  |  |
| Why snippets live in one JSON file instead of a SQLite database |  |  |
| The `-dir` flag is now called `-path` |  |  |
| `Store` must not be used from multiple goroutines at once |  |  |
| A worked example: add your first snippet, then print it back |  |  |
| The idea of someday adding tag-based search |  |  |

Behavior I think should *change* (proposed code changes I deliberately did
not make, and did not document as if made):

## Reflection

Which of my four artifacts will rot first, and what would keep it honest:
