package core

type File struct {
	Path   string `json:"path"`
	Remote string `json:"remote"`
	ACL    string `json:"acl"`
}

type ConfigFile struct {
	Bucket string `json:"bucket"`
	Files  []File `json:"files"`
}
