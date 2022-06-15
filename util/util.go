package util

type VersionType struct {
	Major    uint
	Minor    uint
	Revision uint
}

var Version = VersionType{
	Major:    0,
	Minor:    1,
	Revision: 0,
}
