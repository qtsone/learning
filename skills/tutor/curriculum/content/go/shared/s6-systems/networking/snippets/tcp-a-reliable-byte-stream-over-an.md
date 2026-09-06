In Go: `conn.Read(buf)` returning fewer bytes than you asked for is not an
error — it is the normal case. When you need exactly N bytes, loop, or use
`io.ReadFull`, which does the loop for you and returns
`io.ErrUnexpectedEOF` if the stream dies partway.
