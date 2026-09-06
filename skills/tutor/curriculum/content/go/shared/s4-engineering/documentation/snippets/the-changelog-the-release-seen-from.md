**In Go:** modules take that contract literally — versions are git tags like
`v0.2.0`, and from v2 onward the major version becomes part of the module
path (`example.com/mod/v2`), which is how the toolchain can promise that
upgrading within a major version never breaks your build's import contract.
