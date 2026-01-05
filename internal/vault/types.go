package vault

type System struct {
	Slug   string
	Name   string
	Class  string
	Titles int
}

type ROMEntry struct {
	ID     int
	Title  string
	System string
}
