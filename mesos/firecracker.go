package mesos

import (
	"strconv"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func (e *Firecracker) getFirecrackerConfig(vmmID string) firecracker.Config {
	socket := "/tmp/" + vmmID + ".socket"

	vcpu, _ := strconv.Atoi(e.Settings["FIRECRACKER_VCPU"])
	mem, _ := strconv.Atoi(e.Settings["FIRECRACKER_MEM_MB"])

	return firecracker.Config{
		SocketPath:      socket,
		KernelImagePath: e.Settings["FIRECRACKER_WORKDIR"] + "/vmlinux",
		LogLevel:        "DEBUG",
		LogPath:         "/dev/stdout",
		Drives: []models.Drive{{
			DriveID:      firecracker.String("1"),
			PathOnHost:   firecracker.String("/tmp/" + vmmID + "-rootfs.ext4"),
			IsRootDevice: firecracker.Bool(true),
			IsReadOnly:   firecracker.Bool(false),
			RateLimiter: firecracker.NewRateLimiter(
				// bytes/s
				models.TokenBucket{
					OneTimeBurst: firecracker.Int64(1024 * 1024), // 1 MiB/s
					RefillTime:   firecracker.Int64(500),         // 0.5s
					Size:         firecracker.Int64(1024 * 1024),
				},
				// ops/s
				models.TokenBucket{
					OneTimeBurst: firecracker.Int64(100),  // 100 iops
					RefillTime:   firecracker.Int64(1000), // 1s
					Size:         firecracker.Int64(100),
				}),
		}},
		NetworkInterfaces: []firecracker.NetworkInterface{{
			// Use CNI to get dynamic IP
			CNIConfiguration: &firecracker.CNIConfiguration{
				NetworkName: "fcnet",
				IfName:      "veth0",
			},
		}},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(int64(vcpu)),
			MemSizeMib: firecracker.Int64(int64(mem)),
		},
	}
}
