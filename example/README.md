# Profiling Instrumentation Example

A Go pprof profiling instrumentation.

These instructions expect you have [docker-compose](https://docs.docker.com/compose/) installed.
The server generates span information to stdout.

Bring up the `server` and `client` services to run the example:

```sh
docker-compose up --build
```

Shut down the services when you are finished with the example:

```sh
docker-compose down
```
