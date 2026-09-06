In Go the injected dependencies are interface (or func) fields, and the
zero-cost fake for anything that writes text is `io.Writer` — pass
`os.Stdout` in production and a `bytes.Buffer` in tests, exactly the S3 io
philosophy paying rent.
