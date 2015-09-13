Package vfs defines a basic filesystem abstraction layer that can open files
and walk trees.

Package vfs/bindata is like a lot of existing tools for packaging static assets
into your binary. The difference is that the resulting filesystem is walkable
from root or any subdir.

Command cmd/bindata takes one or more directories and generates Go source files
using package vfs/bindata. They are generated under a new package
`bindata_files` in the current package to avoid namespace clobbering. Use it
with `go generate` by adding a comment to any source file:

```go
//go:generate bindata -skip=*.sw[nop] public templates files
```
