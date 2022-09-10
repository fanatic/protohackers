# Protohackers

Solutions to the [Protohackers](https://protohackers.com/) programming challenge in Go.

Aiming for good observability as well as correct handling of edge cases.

Tested via GitHub Actions. Deployed to Fly.io.

## Level 0: Smoke Test

Package `smoketest` implements a TCP Echo Service from RFC 862.

## Level 1: Prime Time

Package `primetime` implements a JSON request/response service that responds to isPrime methods.

## Level 2: Means to an End

Package `meanstoanend` implements a TCP storage service for prices, returning means.
