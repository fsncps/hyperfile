package sidebar

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fsncps/hyperfile/src/internal/utils"
	"github.com/shirou/gopsutil/v4/disk"
)

// getDeviceDirectories returns physical/removable disk mount points with usage percentage.
func getDeviceDirectories() []directory {
	parts, err := disk.Partitions(false)
	if err != nil {
		slog.Error("Error while getting external media: ", "error", err)
		return nil
	}
	var disks []directory
	for _, p := range parts {
		if shouldListDisk(p.Mountpoint) {
			usage := ""
			if u, err := disk.Usage(p.Mountpoint); err == nil {
				usage = fmt.Sprintf("%d%%", int(u.UsedPercent))
			}
			disks = append(disks, directory{
				Name:     diskName(p.Mountpoint),
				Location: diskLocation(p.Mountpoint),
				usage:    usage,
			})
		}
	}
	return disks
}

// getNetworkDirectories returns mounted network filesystems (NFS, CIFS, SSHFS, etc.).
func getNetworkDirectories() []directory {
	parts, err := disk.Partitions(true)
	if err != nil {
		slog.Error("Error while getting network mounts", "error", err)
		return nil
	}
	var dirs []directory
	for _, p := range parts {
		if isNetworkFstype(p.Fstype) {
			dirs = append(dirs, directory{
				Name:     diskName(p.Mountpoint),
				Location: diskLocation(p.Mountpoint),
			})
		}
	}
	return dirs
}

func isNetworkFstype(fstype string) bool {
	switch fstype {
	case "nfs", "nfs4", "cifs", "smbfs", "fuse.sshfs", "fuse.samba", "glusterfs", "davfs", "afp":
		return true
	}
	return false
}

func shouldListDisk(mountPoint string) bool {
	if runtime.GOOS == utils.OsWindows {
		return true
	}

	if mountPoint == "/" {
		return true
	}

	if strings.HasPrefix(mountPoint, "/Volumes/.timemachine") {
		return false
	}

	return strings.HasPrefix(mountPoint, "/mnt") ||
		strings.HasPrefix(mountPoint, "/media") ||
		strings.HasPrefix(mountPoint, "/run/media") ||
		strings.HasPrefix(mountPoint, "/Volumes")
}

func diskName(mountPoint string) string {
	if runtime.GOOS == utils.OsWindows {
		return mountPoint
	}
	return filepath.Base(mountPoint)
}

func diskLocation(mountPoint string) string {
	if runtime.GOOS == utils.OsWindows {
		return filepath.Join(mountPoint, "\\")
	}
	return mountPoint
}
