> **In Go:** Go does *not* perform tail-call optimization — every recursive
> call consumes a frame, tail position or not. That's why Go code iterates
> over linear structures and typically saves recursion for nested ones.
