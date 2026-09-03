package memlab

import "strconv"

// AppendSample appends one metrics line, "<name> <value>\n", to dst and
// returns the extended slice. It allocates only if dst lacks capacity —
// the same contract as strconv.AppendInt, which does the digit work.
func AppendSample(dst []byte, name string, value int64) []byte {
	dst = append(dst, name...)
	dst = append(dst, ' ')
	dst = strconv.AppendInt(dst, value, 10)
	return append(dst, '\n')
}
