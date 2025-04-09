.PHONY: profile
profile:
	go tool pprof http://localhost:8088/debug/pprof/profile?seconds=60

.PHONY: driver
driver:
	docker plugin install grafana/loki-docker-driver:3.3.2-amd64 --alias loki --grant-all-permissions