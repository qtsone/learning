# Strings, Bytes & Runes

> `go.basics.strings-runes` · ~3-4h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Explain how Go strings store UTF-8 bytes and why `len(s)` is not the character count.
- Predict the difference between indexing a string (byte) and ranging over it (rune).
- Use the `strings` package (`Split`, `Join`, `Contains`, `TrimSpace`, etc.) to solve text-processing tasks.
- Use `strings.Builder` for repeated concatenation and explain why `+=` in a loop is wasteful.

## A string is a sequence of bytes

You have used strings since your very first `fmt.Println`. Time to look inside
one. A Go string is a read-only sequence of **bytes** — the 8-bit numbers that
computers actually store. `len(s)` counts those bytes:

```go
len("hello")   // 5
len("héllo")   // 6 — six bytes for five characters?
```

For English letters, digits, and punctuation, one character fits in one byte
and the distinction is invisible. The moment an `é`, a `€`, or a `語` appears,
bytes and characters part ways — and any code that assumed they were the same
quietly breaks. This lesson is about never writing that code.

## Unicode and UTF-8, briefly

**Unicode** is a catalog that assigns every character in every writing system
a number, called a **code point**: `A` is 65, `é` is 233, `語` is 35486. Go's
name for a code point is a **rune**.

**UTF-8** is the encoding — the rule for writing those numbers down as bytes.
It is variable-width: a code point takes 1 to 4 bytes. The first 128 code
points (plain English text) take exactly one byte each, `é` takes two, `語`
takes three. That is where the sixth byte of `"héllo"` lives: five characters,
but `é` occupies two bytes.

Go source files are UTF-8, and a string literal stores exactly its UTF-8
bytes. So a Go string is best read as "UTF-8-encoded text" — bytes on the
inside, characters only after decoding.

## Indexing gives bytes, ranging gives runes

Index a string like you index a slice and you get a **byte** — a `uint8`
number, not a character:

```go
s := "héllo"
s[0]   // 104 — the byte for 'h'
s[1]   // 195 — the FIRST HALF of é's two bytes, not é
```

But `range` over a string does something special: it decodes UTF-8 as it
walks, handing you one **rune** at a time. The index is the byte offset where
the rune starts, so watch it jump:

```go
for i, r := range "héllo" {
	fmt.Println(i, string(r))
}
// 0 h
// 1 é
// 3 l   ← the index skips 2: é used bytes 1 and 2
// 4 l
// 5 o
```

`rune` is just another name for `int32`, and `byte` another name for `uint8` —
both are numbers. A character in single quotes is a **rune literal**: `'A'` is
the rune 65. And note what you can't do: strings are immutable, so
`s[0] = 'H'` does not compile. To "change" a string you build a new one.

## Converting: []rune, []byte, and back

When you need character-positional work — count them, reverse them, take the
first one — convert the whole string:

```go
runes := []rune("héllo")   // 5 elements, one per character
len(runes)                 // 5
string(runes)              // back to "héllo"
```

The conversion copies and decodes the whole string, so reach for it when
positions matter, not by reflex. (For counting alone, the standard library has
`utf8.RuneCountInString(s)` in `unicode/utf8`.)

One classic trap: converting a *number* to a string. `string(rune(65))` is
`"A"` — the character with that code point — not `"65"`. When you want the
digits, use `fmt.Sprint(65)`.

## The strings package

Most text tasks need no byte-level thinking at all, because the standard
library's `strings` package has done it for you. Since strings are immutable,
every function **returns a new string** and leaves the input alone:

```go
strings.Contains("seafood", "foo")        // true
strings.Split("a,b,c", ",")               // []string{"a", "b", "c"}
strings.Join([]string{"a", "b"}, "-")     // "a-b"
strings.Fields(" go  is fun ")            // []string{"go", "is", "fun"}
strings.TrimSpace("  hi \n")              // "hi"
strings.ToLower("Héllo")                  // "héllo" — rune-aware, é included
strings.ToUpper("héllo")                  // "HÉLLO" — its mirror image
strings.ReplaceAll("co-op", "-", " ")     // "co op"
```

`Split` and `Fields` differ usefully: `Split` cuts at every occurrence of a
separator you name (and keeps empty pieces), while `Fields` splits around any
run of whitespace and never returns empties. Before writing a text loop by
hand, skim [pkg.go.dev/strings](https://pkg.go.dev/strings) — the function you
are about to write probably exists.

## Building strings: why += in a loop is wasteful

Because strings are immutable, `s += piece` cannot extend `s` in place — it
allocates a brand-new string and copies *everything* into it. Do that in a
loop and each pass re-copies all the text accumulated so far: for n pieces
that is 1+2+…+n copies of ever-growing strings. It works, but the cost grows
with the *square* of the input.

`strings.Builder` is the fix: an append-friendly buffer, like the slices you
`append` to. Write into it as often as you like, then take the finished string
once:

```go
var b strings.Builder
for _, word := range words {
	b.WriteString(word)
	b.WriteRune(' ')
}
result := b.String()
```

Rule of thumb: a couple of `+` for a handful of known pieces is fine; a loop
that grows a string calls for a `Builder` — or for `strings.Join`, when what
you have is already a slice.

## Exercise

Open [`exercise/`](exercise/) — a Go module with package `text` in two files
(a library package, tested directly, like in the packages lesson):

- `text.go` — five functions with `TODO`s for you.
- `text_test.go` — the specification. **Read it first**; the non-ASCII cases
  are the point of this lesson.

Acceptance criteria:

1. `CountRunes(s)` returns the number of characters: `CountRunes("héllo")` is
   5 even though `len("héllo")` is 6.
2. `Reverse(s)` reverses the characters, keeping multi-byte ones intact:
   `Reverse("héllo")` is `"olléh"`, never mangled bytes.
3. `CleanFields(csv)` splits on commas, trims the spaces around each entry,
   and drops entries that are empty after trimming: `" a ,, b , "` becomes
   `["a", "b"]`.
4. `Slug(title)` lower-cases the title and joins its words with hyphens:
   `Slug("  Go Is Fun ")` is `"go-is-fun"`.
5. `Initials(name)` returns the upper-cased first character of each word,
   each followed by a period: `Initials("émile zola")` is `"É.Z."` — built
   with a `strings.Builder`, and rune-aware (`word[0]` mangles the é).
6. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green, one function at a time.

## Further reading

- [Go blog — Strings, bytes, runes and characters in Go](https://go.dev/blog/strings)
- [pkg.go.dev — the strings package](https://pkg.go.dev/strings)
- [pkg.go.dev — strings.Builder](https://pkg.go.dev/strings#Builder)
- [pkg.go.dev — unicode/utf8](https://pkg.go.dev/unicode/utf8)
