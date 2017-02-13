package watch

import "strings"

var DefaultIgnore = []string{
	// hidden and temporary files
	".*", "~*", "*~",
	// object files
	"*.[ao]", "*.so", "*.obj",
	// log files
	"*.log",
	// temporary Go files
	"*.test", "*.prof",
	// windows binary files
	"*.exe", "*.dll",
}

type Globs struct {
	NoDefault  bool
	Default    []string
	Additional []string
}

func (globs *Globs) All() []string {
	if globs.NoDefault {
		return globs.Additional
	}

	return append(append([]string{}, globs.Default...), globs.Additional...)
}

func (globs *Globs) String() string {
	return strings.Join(globs.All(), ";")
}

func (globs *Globs) Set(value string) error {
	values := strings.Split(strings.Replace(value, ":", ";", -1), ";")
	globs.Additional = append(globs.Additional, values...)
	return nil
}
