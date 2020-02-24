package jsreload

import (
	"io/ioutil"
	"regexp"
)

var (
	rxPackage = regexp.MustCompile(`\bpackage\s*\(\s*"([^"]+)"\s*,\s*function`)
	rxDepends = regexp.MustCompile(`\bdepends\s*\(\s*"([^"]+)"\s*\)`)
)

func extractPackageInfo(filename string) (pkgname string, depends []string) {
	depends = []string{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	if m := rxPackage.FindStringSubmatch(string(data)); len(m) > 0 {
		pkgname = m[1]
	}
	for _, dependency := range rxDepends.FindAllStringSubmatch(string(data), -1) {
		depends = append(depends, dependency[1])
	}
	return
}
