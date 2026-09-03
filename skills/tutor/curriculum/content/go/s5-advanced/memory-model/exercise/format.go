package memlab

import "fmt"

// AppendSample appends one metrics line, "<name> <value>\n", to dst and
// returns the extended slice — the append-style API you met in stdlib-io
// and throughout the standard library (strconv.AppendInt, AppendFormat).
//
// TODO: this version is correct but heap-allocates on every call — every
// argument passed to fmt escapes into its ...any parameter, and Sprintf
// builds a brand-new string. Rebuild the line with append and strconv so it
// makes zero allocations when dst has capacity (criterion 4). Record the
// escape-analysis output before and after in NOTES.md.
func AppendSample(dst []byte, name string, value int64) []byte {
	return append(dst, fmt.Sprintf("%s %d\n", name, value)...)
}
