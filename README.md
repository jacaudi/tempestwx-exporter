# Tempest Weather Station Exporter

This is a Prometheus/OpenMetrics exporter for [Tempest weather
stations](https://weatherflow.com/tempest-home-weather-system/).

This tool listens for [Tempest UDP broadcasts](https://weatherflow.github.io/Tempest/api/udp.html) and forwards metrics
to a Prometheus push gateway.

## Quickstart

Container images are available at [GitHub
Container Registry](https://github.com/jacaudi/tempestwx-exporter/pkgs/container/tempestwx-exporter).

```bash
$ docker run -it --rm --net=host \
  -e PUSH_URL=http://victoriametrics:8429/api/v1/import/prometheus \
  ghcr.io/jacaudi/tempestwx-exporter

2023/07/06 20:18:55 pushing to "0.0.0.0" with job name "tempest"
2023/07/06 20:18:55 listening on UDP :50222
```

Note that `--net=host` is used here because UDP broadcasts are link-local and therefore cannot be received from typical
(routed) container networks.

## Exporter configuration

Minimal, via environment variables:

* `PUSH_URL`: the URL of the [Prometheus Pushgateway](https://github.com/prometheus/pushgateway) or other [compatible
  service](https://docs.victoriametrics.com/?highlight=exposition#how-to-import-data-in-prometheus-exposition-format)

* `JOB_NAME`: the value for the `job` label, defaulting to `"tempest"`

## Source Credit

- [tempest-exporter](https://github.com/willglynn/tempest_exporter) - Started as a fork of this project.
