# STALEPROXY

staleproxy is a simple proxy that serves stale content when available and does an asynchronous request afterwards to update the cache.

## USAGE

set env vars:

PROXY_BASE_URL: <URL_TO_PROXY_TO_WITH_TRAILING_SLASH>
REWRITE_FROM: <URL_TO_REWRITE_FROM>
REWRITE_TO: <URL_TO_REWRITE_TO>

