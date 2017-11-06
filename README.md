Foreman
---
A CLI that keeps the miners mining.

This tool tails the log of your claymore miner and reboots the machine when one of the GPUs has crashed.


Getting Started
---

1. Download the binary from this repo and upload to your rig (script assumes /home/ethos/)

2. Run the install command to create an upstart job and kick off the process
```
    sudo foreman install
```
