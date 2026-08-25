//go:build !windows

package storage

import "os"

func isLinkOrReparse(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
