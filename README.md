# Orbit Telemetry Orchestrator

Orbit is a standard-library Go service for coordinating device telemetry,
collection policies, anomaly detection, incident response, and evidence
packaging in edge operations.

## Run

```bash
go run ./cmd/orbitd
```

The default HTTP address is `:8181`. Set `ORBIT_ADDR` to override it.

## Flow

1. Register a device with `POST /v1/devices`.
2. Add a telemetry stream with `POST /v1/streams`.
3. Publish a collection policy with `POST /v1/policies`.
4. Ingest samples with `POST /v1/telemetry`.
5. Review anomalies and incidents through `/v1/anomalies` and `/v1/incidents`.
6. Execute a response action and close an incident after evidence is attached.

The service intentionally uses an in-memory repository so the complete domain
flow can be run without external infrastructure.
