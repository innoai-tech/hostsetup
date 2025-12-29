package host

type Package struct {
	// Name 包名
	Name string `json:"name"`
	// Version 包完整版本
	Version string `json:"version,omitzero"`
	// DownloadURL 包直接下载地址
	DownloadURL string `json:"downloadURL,omitzero"`
	// DownloadOnly 包只作为依赖下载，执行时不需要安装
	DownloadOnly bool `json:"downloadOnly,omitzero"`
	// Args 安装时额外参数
	Args []string `json:"args,omitzero"`
}
