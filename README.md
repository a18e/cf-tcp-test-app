# go-fail
HTTP Server that fails when needed

## What it does
The program offers a `/fail` endpoint that allows you to set a probability `p` that the HTTP request will fail.
"failing" in this sense means the client does not receive a HTTP response at all. It does NOT mean returning a status 500 or whatever
as that technically does not "fail" on a TCP level. The failure actually crashes the app as this is the cheapest way to abort the connection.
It relies on an outer health check to resurrect the app.

## Why is it useful?
The primary use is for load balancing backend resiliency tests.
E.g. what happens if 1 out of 3 instances fail? Does the load balancer retry on another one? Will it disable the broken instance for a while? How many retries does it? etc.

## How to build
```
go build
```

## How to run
You can either just run the binary or push it to CloudFoundry with the given `manifest.yml`

## How to use
You can provide a URL parameter `p` to set the probability of the app to fail.
```
curl http://myhost:8080/fail?p=25
```
e.g. this would fail in 25% of requests.
