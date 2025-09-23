# FaultLine

All-in-one failure testing for APIs and Databases.

## Commands


## Quick start

1. Configure your scenarios in `faultline.yaml`.
2. Start API proxy:

	 faultline start-api -c faultline.yaml -p 8080

3. Start DB proxies:

	 faultline start-db -c faultline.yaml

When you launch the CLI, you'll see a friendly ASCII banner. To hide it, set `FAULTLINE_NO_BANNER=1` in your environment.

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
