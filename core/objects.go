package core

type File struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Remote  string `json:"remote"`
	ACL     string `json:"acl"`
}

type ConfigFile struct {
	Bucket    string `json:"bucket"`
	Files     []File `json:"files"`
	GlobalACL string `json:"acl"`
}
