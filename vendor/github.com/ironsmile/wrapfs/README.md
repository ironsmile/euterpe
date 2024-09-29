# WrapFS

`wrapfs` is a Go module which provides wrappers around `fs.FS`.

## ModTimeFS

This wrapper makes sure calls to [fs.FileInfo.Stat](https://pkg.go.dev/io/fs#FileInfo.Stat)
and [fs.DirEntry.Info](https://pkg.go.dev/io/fs#DirEntry.Info) always return non-zero
[ModTime()](https://pkg.go.dev/io/fs#FileInfo.ModTime). In case the wrapped file or entry
return non-zero modification time it stays unchanged. In case they return a zero
modification time then a static mod time will be used instead.

Usage:

```go
var someFs fs.FS = getFS()
modTimeFS := wrapfs.WithModTime(someFS, time.Now())
http.FileServer(http.FS(modTimeFS))
```

This is especially helpful for `embed.FS` file systems when used in conjecture with
`http.FileServer`. When `wrapfs.WithModTime` is used in this case the HTTP server will
be able to handle caches which utilize the "Last-Modified" HTTP headers.

## Known Limitations

If new interfaces are added in `fs` they will not be exposed as implemented by the
wrappers until the library has been patched.
