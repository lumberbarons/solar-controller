#!/bin/bash

# Create log file if it doesn't exist
if [ ! -f /var/log/solar-controller.log ]; then
    touch /var/log/solar-controller.log
    chmod 644 /var/log/solar-controller.log
fi

systemctl daemon-reload
systemctl enable solar-controller