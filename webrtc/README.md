# Pion WebRTC service

** WIP **

## Build

```
go build
./webrtc -h
```

```
Usage of ./webrtc:
      --log_level string        Set the log level, eg. info, warn, debug or error (default "info")
      --media_addr string       Websocket for consuming media (default "127.0.0.1:8090")
      --messaging_addr string   Websocket for consuming SDP messaging (default "127.0.0.1:8091")
```

## Configuration

The application can be configured through CLI options or through environment (using the `WEBRTC_` namespace)

eg. cli option

```
./webrtc --log_level debug --messaging_addr 127.0.0.1:8888
```

or through env

```
WEBRTC_LOG_LEVEL=debug WEBRTC_MESSAGING_ADDR=127.0.0.1:8888 ./webrtc
```
