**In Go:** the conventions you've absorbed since S1 are these criteria
applied. `MixedCaps`, never underscores; short receiver names (`func (s
*Server)`) because the receiver's scope is one function and its type is a line
away; `err` and `ok` as fixed idioms; and no package stutter — a name is
always read *with* its package qualifier, so `report.Builder`, not
`report.ReportBuilder`. Effective Go's advice that a name's length should
match its scope is exactly the last criterion above.
