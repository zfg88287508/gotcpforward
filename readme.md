# Go tcp forward

This application can forward tcp connection to upstream.
## build

```
CGO_ENABLED=0 go build -v -a -ldflags ' -s -w  -extldflags "-static"' .
```

## use

```
  -l string
    	listen host:port
  -r string
    	remote host:port

```

Example use, forward local port to remote server port

```
./gotcpforward -l :3306 -r 10.1.23.43:3316
```
