# epever controller

```bash
docker run -d --name solar --device /dev/ttyXRUSB0 -p 8080:8080 --restart=unless-stopped \
  --net=telegraf --env SERIAL_PORT=/dev/ttyXRUSB0 ghcr.io/lumberbarons/epever_controller:v0.3.0
```