# testing

## setup

```bash
pip3 install modbus_tk
```

## run

```bash
socat -d -d -d pty,raw pty,raw
./simulator.py <port-1>
go run main.go --config ./test/config.yaml
```
