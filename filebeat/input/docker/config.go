package docker

var defaultConfig = config{
	Partial: true,
	Containers: containers{
		IDs:           []string{},
		Path:          "/var/lib/docker/containers",
		Stream:        "all",
		PodId:         "",
		KubeletPath:   "/var/lib/kubelet/pods",
		EmptyDirPaths: []string{},
	},
}

type config struct {
	Containers containers `config:"containers"`

	// Partial configures the prospector to join partial lines
	Partial bool `config:"combine_partials"`
}

type containers struct {
	IDs  []string `config:"ids"`
	Path string   `config:"path"`

	// Stream can be all,stdout or stderr
	Stream string `config:"stream"`

	PodId         string   `config:"pod_id"`
	KubeletPath   string   `config:"kubelet_path"`
	EmptyDirPaths []string `config:"empty_dir_paths"`
}
