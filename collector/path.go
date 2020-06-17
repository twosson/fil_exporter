package collector

import (
	"github.com/prometheus/procfs"
	"gopkg.in/alecthomas/kingpin.v2"
	"path/filepath"
	"strings"
)

var (
	ProcPath = kingpin.Flag("path.procfs", "procfs mountpoint.").Default(procfs.DefaultMountPoint).String()
	SysPath = kingpin.Flag("path.sysfs", "sysfs mountpoint.").Default("/sys").String()
	RootFsPath = kingpin.Flag("path.rootfs", "rootfs mountpoint.").Default("/").String()
)

func ProcFilePath(name string) string {
	return filepath.Join(*ProcPath, name)
}

func SysFilePath(name string) string {
	return filepath.Join(*SysPath, name)
}

func RootfsFilePath(name string) string {
	return filepath.Join(*RootFsPath, name)
}

func RootfsStripPrefix(path string) string {
	if *RootFsPath == "/" {
		return path
	}
	stripped := strings.TrimPrefix(path, *RootFsPath)
	if stripped == "" {
		return "/"
	}
	return stripped
}

