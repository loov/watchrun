package watch

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func IgnoreAll(filters ...Filter) Filter {
	if len(filters) == 1 {
		return filters[0]
	}
	return func(path string, info os.FileInfo) bool {
		for _, filter := range filters {
			if filter(path, info) {
				return true
			}
		}
		return false
	}
}

func IgnoreExtensions(exts ...string) Filter {
	list := []string{}
	for _, ext := range exts {
		list = append(list, cname(ext))
	}

	return func(path string, info os.FileInfo) bool {
		ext := cname(filepath.Ext(path))
		for _, item := range list {
			if ext == item {
				return true
			}
		}
		return false
	}
}

func IgnoreNameSuffixed(suffixes ...string) Filter {
	list := []string{}
	for _, suffix := range suffixes {
		list = append(list, cname(suffix))
	}

	return func(path string, info os.FileInfo) bool {
		name := cname(filepath.Base(path))
		for _, suffix := range list {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
		return false
	}
}

func IgnoreNamePrefixed(prefixes ...string) Filter {
	list := []string{}
	for _, prefix := range prefixes {
		list = append(list, cname(prefix))
	}

	return func(path string, info os.FileInfo) bool {
		name := cname(filepath.Base(path))
		for _, prefix := range list {
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
		return false
	}
}

func cname(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(name)
	}
	return name
}
