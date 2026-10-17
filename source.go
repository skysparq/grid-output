package gridoutput

import "strings"

// SourceName is the isobands' "source" property for grids from the given system and subsystem.
func SourceName(system, subsystem string) string {
	if system == `GDAS` || system == `ERA5` {
		return strings.ToLower(system)
	}
	return strings.ToLower(subsystem)
}
