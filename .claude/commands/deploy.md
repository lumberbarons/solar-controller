---
description: Deploy the solar controller application to a remote server
allowed-tools: Bash(make deploy:*)
---

## Instructions

1. Ask the user for the remote host in the format `user@host` (e.g., `jim@100.90.141.9`)
2. Run `make deploy REMOTE_HOST=<user@host>` with the provided remote host
3. Show the deployment progress and results to the user
4. If deployment fails, help diagnose the issue

## Notes

- The deploy target automatically builds the Linux ARM64 binary before deployment
- The binary is copied to the remote server's home directory, then moved to `/usr/bin`
- The service is restarted automatically after installation
- SSH key authentication should be set up for the remote host
