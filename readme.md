# Go tcp forward

This application can forward tcp connection to upstream.
## build

```
go build -v -a -ldflags ' -s -w -static -extldflags "-static"' .
```
