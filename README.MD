# Mesos Firecracker Executor

This executer will deploy a microvm per task, and run the mesos task command inside of it.

## Notes

If the custom executer would be running under the mesos containerizer. The default isolation configuration in mesos
is to tight. For the first shot, change the isolation to:

```
MESOS_ISOLATION=filesystem/linux,docker/runtime
```

My meso-compose framework will use the docker containerizer. These should work out of the box.
