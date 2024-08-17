# OS API

An API for managing Linux systems, responsible for ensuring that the system's
configuration matches the desired state.

## API

The client and server components are generated from an OpenAPI spec.

## Usage

TODO

## Endpoints

WIP - Not exactly sure how to lay this out yet.

* /hardware/
* /hardware/disks/
* /networking/
* /services/
* /services/cron/
* /services/dns/
* /services/ntp/
* /system/
* /system/health/
* /system/health/ping - needs moved
* /system/logs/

# Guiding Principles

TODO

## Testing

Install dependencies:

```bash
task deps
```

To execute tests:

```bash
task test
```

Auto format code:

```bash
task fmt
```

List helpful targets:

```bash
task
```

## License

The [MIT][] License.

[MIT]: LICENSE
