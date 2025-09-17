# Mesos Firecracker Executor

This executer will deploy a microvm per task, and run the mesos task command inside of it.


## variables

| Variable | Default | Description |
| --- | --- | ---- |
| FIRECRACKER_AGENT_PORT | 8085 | The port there the vmm-agent is listening |
| FIRECRACKER_PAYLOAD_FILE |  | Payload file for the case the microvm have to run a specific job |
| FIRECRACKER_WORKDIR | /mnt/mesos/sandbox | The directory where the executor will find rootfs, vmlinux and the payload |
| FIRECRACKER_VCPU | 1 | The amount of vCPU for the microvm |
| FIRECRACKER_MEM_MB | 256 | The amount of memory for the microvm | 


## Notes

If the custom executer would be running under the mesos containerizer. The default isolation configuration in mesos
is to tight. For the first shot, change the isolation to:

```
MESOS_ISOLATION=filesystem/linux,docker/runtime
```

My mesos-compose framework will use the docker containerizer. These should work out of the box.

## Build tab Interface for networking

To get access into the microvm, we have to create a tap network interface on our host machine:

```bash
ip tuntap add tap0 mode tap
ip addr add 192.168.127.1/24 dev tap0
ip link set tap0 up
```