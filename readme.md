# Go tcp forward

This application can forward tcp connection to upstream.
## build

```
CGO_ENABLED=0 go build -v -a -ldflags ' -s -w  -extldflags "-static"' .
```
