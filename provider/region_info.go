package provider

import "sort"

type RegionInfo struct {
	Name        string
	Description string
}

// sorts the given slice of cloud provider structs
// in ascending order of name
func sortRegions(regions []RegionInfo) {
	sort.Sort(&regionSorter{regions})
}

// sorter struct contains the slice of providers to
// be sorted and implements the sort.Interface
type regionSorter struct {
	regions []RegionInfo
}

// Len is part of sort.Interface.
func (cps *regionSorter) Len() int {
	return len(cps.regions)
}

// Swap is part of sort.Interface.
func (cps *regionSorter) Swap(i, j int) {
	cps.regions[i], cps.regions[j] = cps.regions[j], cps.regions[i]
}

// Less is part of sort.Interface. It is implemented
// by calling the "by" closure in the sorter.
func (cps *regionSorter) Less(i, j int) bool {
	return cps.regions[i].Name < cps.regions[j].Name
}
