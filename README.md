# go-tcp-test
Test app for Cloud Foundry app lifecycle & health check debugging.


## What it does
The app offers various endpoints that allow for fine control over its health state & responses on TCP level.

It also logs every request it receives, plus various lifecycle events.

### `/togglehealth`:
Toggles App Health State and returns new app health state.
### `/health`:
Returns a `200` if the app is healthy, drops/closes the connection otherwise.   

### `/drop`:
Always drops the Connection. Results in Gorouter error-code `502` with `x_cf_routererror:"endpoint_failure (EOF (via idempotent request))"`.

### `/always_up`:
Always returns a `200` response even when the app is unhealthy.

### All others
Returns a `200` if the app is healthy, drops/closes the connection otherwise.
Also writes the HTTP Method into the response

## How to build
```
go build
```

## How to run
Either run the binary or push it to CloudFoundry with the given `manifest.yml`.

Use `health-check-type: process` for [process healthchecks](https://docs.cloudfoundry.org/devguide/deploy-apps/healthchecks.html#types), or the following for HTTP health checks:
```yaml
    health-check-type: http
    health-check-http-endpoint: /health
```

The following environment variables affect the apps behaviour and can be specified in the manifest (see example).


### `INITIAL_HEALTH`
If `INITIAL_HEALTH: false` is specified, the app will start with failing Health endpoint (i.e. `cf push` will fail)

### Customizable Wait Periods 

```mermaid
flowchart LR
    start["Process started"]--->|START_DELAY| listens["listener starts"] -.-|App runs| crash["SIGTERM received"]--->|DRAIN_DELAY| drain["listener stopped"]--->|STOP_DELAY| stop["Process stopped"]
    
```

`START_DELAY`, `DRAIN_DELAY` and `STOP_DELAY` can be set to durations, e.g. "10s", "5m".

## How to use
After pushing the app to CF, use `cf logs go-tcp-test` to see the detailed log output of every request/health-check/event.
## References
- [Health Check Lifecycle in CF Docs](https://docs.cloudfoundry.org/devguide/deploy-apps/healthchecks.html#healthcheck-lifecycle)

## App Lifecycle (WIP)


### App Crash: 
```mermaid
sequenceDiagram
    participant re as route_emitter/NATS
    participant ha as client/HAProxy
    participant gorouter
    participant envoy
    participant executor
    participant hc as healthcheck (binary)
    participant app

    activate app
    activate envoy
    executor ->>+ hc: Start 
    
    loop Every 30s
        hc ->> app: Health Check
        app -->> hc: OK
    end

    note over app: App becomes unhealthy
    
    hc -X app: Health Check (tcp or http)
    opt Improvement Idea?
    hc->>envoy: shutdown
    end
    note over hc: exits (see exit code!)
    hc ->> executor: exit status
    deactivate hc
    
    activate executor
    opt Improvement Idea?
    executor->>envoy: Invalidate Certs
    end
    executor->>app: SIGTERM
    deactivate executor
        
    loop
    alt slow envoy cert invalidation
    ha ->>+ gorouter: request
    gorouter ->>+ envoy: request
    envoy --X- app: request
    gorouter -->>- ha: 502 error (EOF)
    ha ->>+ gorouter: request
    gorouter -->>- ha: 503 error
    else envoy certs invalidated
    ha ->>+ gorouter: request
    gorouter ->>+ envoy: request
    envoy ->>- gorouter: invalid certs
    gorouter -->>- ha: 503 error (invalid certs)
    end
    
    note over gorouter: drops route
    
    loop
    ha ->>+ gorouter: request
    gorouter -->>- ha: 404 error
    end
    re -) app: checks state
    activate re
    re --) gorouter: publish route
    deactivate re
    note over gorouter: adds route
    end
    note over app: App Container stops/is restarted
    
    deactivate app

```

