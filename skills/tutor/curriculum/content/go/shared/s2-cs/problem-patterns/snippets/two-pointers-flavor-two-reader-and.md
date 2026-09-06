> **In Go:** remember from the arrays-and-linked-lists lesson that a slice is
> a small header pointing at a backing array. Writing `nums[w] = …` inside
> the function writes to the *caller's* backing array — that is what "in
> place" means here. The convention is to return the new length `k` and let
> the caller keep working with `nums[:k]`.
