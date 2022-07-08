# Mesos Firecracker Executor

This executer will deploy a microvm per task, and run the mesos task command inside of it.

## How to use

<TODO>


## Notes

Every custom executer would be running under the mesos containerizer. The default isolation configuration in mesos
would be to tight. For the first shot, we should change the isolation to:

```
MESOS_ISOLATION=filesystem/linux,docker/runtime
```

It definitly a todo to find out which isolation we can use together with firecracker. I recommend to run firecracker 
workload only on "special" mesos nodes and forbid all other wordload to run there.

