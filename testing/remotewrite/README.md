# Testing Remote Write

Test Prometheus remote_write locally with VictoriaMetrics.

## Quick Start

```bash
# From the testing/remotewrite/ directory, start VictoriaMetrics:
cd testing/remotewrite
./test-remotewrite.sh

# In another terminal, build and run solar-controller:
make build-backend
./bin/solar-controller -config testing/remotewrite/config.yaml

# Open VictoriaMetrics UI:
open http://localhost:8428/vmui
```

Query your metrics:
```
epever_battery_voltage
{device_id="test-controller-1"}
```

## Cleanup

```bash
docker stop victoriametrics-test
```

That's it!
