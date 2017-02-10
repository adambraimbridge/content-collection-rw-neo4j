package storypackage

type storyPackage struct {
	UUID             string `json:"uuid,omitempty"`
	Items            []item `json:"items,omitempty"`
	PublishReference string `json:"publishReference"`
	LastModified     int64  `json:"lastModified"`
}

type item struct {
	UUID string `json:uuid`
}
