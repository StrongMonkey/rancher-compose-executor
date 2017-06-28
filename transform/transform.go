package transform

import (
	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/libcompose/utils"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/yaml"
	"strconv"
	"strings"
)

const (
	READ_IOPS    = "readIops"
	WRITE_IOPS   = "writeIops"
	READ_BPS     = "readBps"
	WRITE_BPS    = "writeBps"
	WEIGHT       = "weight"
	IMAGE_PREFIX = "docker:"
)

func Transform(serviceConfig *config.ServiceConfig, context *project.Context) (client.LaunchConfig, error) {
	var launchConfig client.LaunchConfig

	launchConfig.Hostname = serviceConfig.Hostname
	launchConfig.Memory = int64(serviceConfig.MemLimit)
	launchConfig.CpuSet = serviceConfig.CPUSet
	launchConfig.CpuShares = int64(serviceConfig.CPUShares)
	launchConfig.MemorySwap = int64(serviceConfig.MemSwapLimit)
	launchConfig.DomainName = serviceConfig.DomainName
	launchConfig.User = serviceConfig.User
	launchConfig.Tty = serviceConfig.Tty
	launchConfig.StdinOpen = serviceConfig.StdinOpen
	if serviceConfig.Image != "" {
		launchConfig.ImageUuid = IMAGE_PREFIX + serviceConfig.Image
	}
	launchConfig.WorkingDir = serviceConfig.WorkingDir
	launchConfig.Environment = mapToMap(serviceConfig.Environment.ToMap())
	launchConfig.Command = strslice.StrSlice(utils.CopySlice(serviceConfig.Command))
	launchConfig.EntryPoint = strslice.StrSlice(utils.CopySlice(serviceConfig.Entrypoint))
	launchConfig.VolumeDriver = serviceConfig.VolumeDriver
	launchConfig.StopSignal = serviceConfig.StopSignal
	launchConfig.DataVolumes = volumes(serviceConfig, *context)
	launchConfig.Ports = serviceConfig.Ports
	launchConfig.Privileged = serviceConfig.Privileged
	launchConfig.Dns = serviceConfig.DNS
	launchConfig.DnsSearch = serviceConfig.DNSSearch
	launchConfig.CapAdd = serviceConfig.CapAdd
	launchConfig.CapDrop = serviceConfig.CapDrop
	launchConfig.Devices = setupDevice(serviceConfig.Devices)
	launchConfig.MemoryReservation = int64(serviceConfig.MemReservation)
	launchConfig.BlkioWeight = int64(serviceConfig.BlkioWeight)
	options, err := toBlkioOptions(serviceConfig)
	if err != nil {
		return client.LaunchConfig{}, err
	}
	launchConfig.BlkioDeviceOptions = options
	launchConfig.CgroupParent = serviceConfig.CgroupParent
	launchConfig.CpuPeriod = int64(serviceConfig.CPUPeriod)
	launchConfig.CpuQuota = int64(serviceConfig.CPUQuota)
	launchConfig.DnsOpt = serviceConfig.DNSOpt
	launchConfig.GroupAdd = serviceConfig.GroupAdd
	launchConfig.MemorySwappiness = int64(serviceConfig.MemSwappiness)
	launchConfig.OomKillDisable = serviceConfig.OomKillDisable
	launchConfig.ShmSize = int64(serviceConfig.ShmSize)
	launchConfig.Tmpfs = tmpfsToMap(serviceConfig.Tmpfs)
	launchConfig.Uts = serviceConfig.Uts
	launchConfig.IpcMode = serviceConfig.Ipc
	launchConfig.Sysctls = mapToMap(serviceConfig.Sysctls)
	launchConfig.OomScoreAdj = int64(serviceConfig.OomScoreAdj)
	launchConfig.Isolation = serviceConfig.Isolation
	launchConfig.Ulimits = toRancherUlimit(serviceConfig.Ulimits)
	launchConfig.SecurityOpt = serviceConfig.SecurityOpt
	launchConfig.PidMode = serviceConfig.Pid
	launchConfig.ReadOnly = serviceConfig.ReadOnly
	launchConfig.ExtraHosts = serviceConfig.ExtraHosts
	launchConfig.LogConfig = toRancherLogOption(serviceConfig.Logging)
	launchConfig.Labels = mapToMap(serviceConfig.Labels)
	return launchConfig, nil
}

func mapToMap(m map[string]string) map[string]interface{} {
	r := make(map[string]interface{})
	for k, v := range m {
		r[k] = v
	}
	return r
}

func volumes(c *config.ServiceConfig, ctx project.Context) []string {
	if c.Volumes == nil {
		return []string{}
	}
	volumes := []string{}
	for _, v := range c.Volumes.Volumes {
		vol := v
		if len(ctx.ComposeFiles) > 0 && !project.IsNamedVolume(v.Source) {
			sourceVol := ctx.ResourceLookup.ResolvePath(v.String(), ctx.ComposeFiles[0])
			vol.Source = strings.SplitN(sourceVol, ":", 2)[0]
		}
		volumes = append(volumes, vol.String())
	}
	return volumes
}

func toBlkioOptions(c *config.ServiceConfig) (map[string]interface{}, error) {
	opts := make(map[string]map[string]uint64)

	blkioDeviceReadBps, err := getThrottleDevice(c.DeviceReadBps)
	if err != nil {
		return nil, err
	}

	blkioDeviceReadIOps, err := getThrottleDevice(c.DeviceReadIOps)
	if err != nil {
		return nil, err
	}

	blkioDeviceWriteBps, err := getThrottleDevice(c.DeviceWriteBps)
	if err != nil {
		return nil, err
	}

	blkioDeviceWriteIOps, err := getThrottleDevice(c.DeviceWriteIOps)
	if err != nil {
		return nil, err
	}

	blkioWeight, err := getThrottleDevice(c.BlkioWeightDevice)
	if err != nil {
		return nil, err
	}

	for _, rbps := range blkioDeviceReadBps {
		_, ok := opts[rbps.Path]
		if !ok {
			opts[rbps.Path] = map[string]uint64{}
		}
		opts[rbps.Path][READ_BPS] = rbps.Rate
	}

	for _, riops := range blkioDeviceReadIOps {
		_, ok := opts[riops.Path]
		if !ok {
			opts[riops.Path] = map[string]uint64{}
		}
		opts[riops.Path][READ_IOPS] = riops.Rate
	}

	for _, wbps := range blkioDeviceWriteBps {
		_, ok := opts[wbps.Path]
		if !ok {
			opts[wbps.Path] = map[string]uint64{}
		}
		opts[wbps.Path][WRITE_BPS] = wbps.Rate
	}

	for _, wiops := range blkioDeviceWriteIOps {
		_, ok := opts[wiops.Path]
		if !ok {
			opts[wiops.Path] = map[string]uint64{}
		}
		opts[wiops.Path][WRITE_IOPS] = wiops.Rate
	}

	for _, w := range blkioWeight {
		_, ok := opts[w.Path]
		if !ok {
			opts[w.Path] = map[string]uint64{}
		}
		opts[w.Path][WEIGHT] = w.Rate
	}

	result := make(map[string]interface{})
	for k, v := range opts {
		result[k] = v
	}
	return result, nil
}

func getThrottleDevice(throttleConfig yaml.MaporColonSlice) ([]*blkiodev.ThrottleDevice, error) {
	throttleDevice := []*blkiodev.ThrottleDevice{}
	for _, deviceWriteIOps := range throttleConfig {
		split := strings.Split(deviceWriteIOps, ":")
		rate, err := strconv.ParseUint(split[1], 10, 64)
		if err != nil {
			return nil, err
		}

		throttleDevice = append(throttleDevice, &blkiodev.ThrottleDevice{
			Path: split[0],
			Rate: rate,
		})
	}

	return throttleDevice, nil
}

func tmpfsToMap(tmpfs []string) map[string]interface{} {
	r := make(map[string]interface{})
	for _, v := range tmpfs {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) == 1 {
			r[parts[0]] = ""
		} else if len(parts) == 2 {
			r[parts[0]] = parts[1]
		}
	}
	return r
}

func toRancherUlimit(ulimits yaml.Ulimits) []client.Ulimit {
	r := []client.Ulimit{}
	for _, u := range ulimits.Elements {
		r = append(r, client.Ulimit{Name: u.Name, Soft: u.Soft, Hard: u.Hard})
	}
	return r
}

func toRancherLogOption(log config.Log) *client.LogConfig {
	var r client.LogConfig
	r.Driver = log.Driver
	r.Config = mapToMap(log.Options)
	return &r
}

func setupDevice(devices []string) []string {
	r := []string{}
	for _, d := range devices {
		tmp := d
		parts := strings.SplitN(d, ":", 3)
		if len(parts) == 2 {
			tmp = tmp + ":rwm"
		}
		r = append(r, tmp)
	}
	return r
}
