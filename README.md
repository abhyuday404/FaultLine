# FaultLine

All-in-one failure testing for APIs and Databases.

## Commands

- start-api: Run the HTTP reverse proxy for API fault injection (latency, errors, flaky).
- start-db: Run the TCP proxy for DB/network fault injection (latency, drops, throttling, refuse).

## Quick start

1. Configure your scenarios in `faultline.yaml`.
2. Start API proxy:

	 faultline start-api -c faultline.yaml -p 8080

3. Start DB proxies:

	 faultline start-db -c faultline.yaml

## Example tcpRules

```
tcpRules:
	- listen: 127.0.0.1:55432
		upstream: localhost:5432
		faults:
			refuseConnections: true
	- listen: 127.0.0.1:55433
		upstream: localhost:5432
		faults:
			latencyMs: 200
			bandwidthKbps: 64
	- listen: 127.0.0.1:55434
		upstream: localhost:5432
		faults:
			dropProbability: 0.2
			resetProbability: 0.1
```

Note: DB command simulates network-level faults. To trigger DB-specific SQLSTATE errors, use a client or helper tool to execute SQL that violates constraints or permissions.
