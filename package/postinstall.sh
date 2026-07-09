#!/bin/bash

# Create dedicated system user for the service
if ! getent passwd solar-controller >/dev/null; then
    useradd --system --no-create-home --home-dir /nonexistent \
        --shell /usr/sbin/nologin solar-controller
fi

# Create log file if it doesn't exist
if [ ! -f /var/log/solar-controller.log ]; then
    touch /var/log/solar-controller.log
    chmod 644 /var/log/solar-controller.log
fi

systemctl daemon-reload
systemctl enable solar-controller
