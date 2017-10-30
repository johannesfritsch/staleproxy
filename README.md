# STALEPROXY

staleproxy is a simple proxy that serves stale content when available and does an asynchronous request afterwards to update the cache.

## USAGE

Set these env vars and run the binary.

| Environment Var   |      Description      |  Sample                |
|-------------------|:----------------------|-----------------------:|
| PROXY_BASE_URL    | The URL to proxy to   | http://google.com/     |
| REWRITE_FROM      |    Rewrite this...    | http://google.com/     |
| REWRITE_TO        | ... to this           | http://localhost:8080/ |