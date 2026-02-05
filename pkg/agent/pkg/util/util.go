package util

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

func IsEnvVarSet(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return value, fmt.Errorf("environment variable %s must be set", key)
	}
	return value, nil
}

// IsFileExist reports whether path exits.
func IsFileExist(fpath string) bool {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return false
	}
	return true
}

// FileInfo describes a configuration file and is returned by fileStat.
type FileInfo struct {
	Uid  uint32
	Gid  uint32
	Mode os.FileMode
	Md5  string
}

// IsConfigChanged reports whether src and dest config files are equal.
// Two config files are equal when they have the same file contents and
// Unix permissions. The owner, group, and mode must match.
// It return false in other cases.
func IsConfigChanged(src, dest string) (bool, error) {
	if !IsFileExist(dest) {
		return true, nil
	}
	d, err := FileStat(dest)
	if err != nil {
		return true, err
	}
	s, err := FileStat(src)
	if err != nil {
		return true, err
	}
	if d.Uid != s.Uid {
		return true, nil
	}
	if d.Gid != s.Gid {
		return true, nil
	}
	if d.Mode != s.Mode {
		return true, nil
	}
	if d.Md5 != s.Md5 {
		return true, nil
	}
	if d.Uid != s.Uid || d.Gid != s.Gid || d.Mode != s.Mode || d.Md5 != s.Md5 {
		return true, nil
	}
	return false, nil
}

func FileStat(name string) (fi FileInfo, err error) {
	if IsFileExist(name) {
		f, err := os.Open(name)
		if err != nil {
			return fi, err
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Printf("failed to close file: %v\n", err)
			}
		}()
		stats, _ := f.Stat()
		fi.Uid = stats.Sys().(*syscall.Stat_t).Uid
		fi.Gid = stats.Sys().(*syscall.Stat_t).Gid
		fi.Mode = stats.Mode()
		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			return fi, fmt.Errorf("failed to copy file data: %v", err)
		}
		fi.Md5 = fmt.Sprintf("%x", h.Sum(nil))
		return fi, nil
	}
	return fi, errors.New("file not found")
}
