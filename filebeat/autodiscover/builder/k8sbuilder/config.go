package k8sbuilder

import (
	"github.com/elastic/beats/libbeat/common"
)

/*
"annotations": {
    // 修改docker标准输出的配置
	"meitu.com.logs/include_lines": "^test1, ^test2",
	"meitu.com.logs/exclude_lines": "^test3, ^test4",
	"meitu.com.logs/multiline.pattern": "^test5",
	"meitu.com.logs/multiline.negate":  "true",

    // 按照容器名称修改的配置.指定名称的配置会覆盖上面的配置
	"meitu.com.logs.contain_name/include_lines": "^test5, ^test6",
	"meitu.com.logs.contain_name/exclude_lines": "^test7, ^test8",

    // 修改扩展目录的配置
	"meitu.com.logs.foobar/extern_path": "/data/*.log",
	"meitu.com.logs.foobar/extern_path_conf.include_lines": "^test6, ^test7",
	"meitu.com.logs.foobar/extern_path_conf.exclude_lines": "^test8, ^test9",
	"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test10",
	"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",
}
*/

func getDefaultDockerConfig(id string) *common.Config {
	rawCfg := map[string]interface{}{
		"type": "docker",
		"containers": map[string]interface{}{
			"ids": []string{
				id,
			},
		},
	}
	cfg, _ := common.NewConfigFrom(rawCfg)
	return cfg
}

func getDefaultExternConfig(path string) *common.Config {
	rawCfg := map[string]interface{}{
		"type": "log",
        "paths": []string {
            path,
        },
	}
	cfg, _ := common.NewConfigFrom(rawCfg)
	return cfg
}

type config struct {
	Key    string         `config:"key"`
	Host   string         `config:"host"`
	DockerAPIVersion string         `config:"docker_api_version"`
	Config *common.Config `config:"config"`
}

func defaultConfig() config {
    /*
	rawCfg := map[string]interface{}{
		"type": "docker",
		"containers": map[string]interface{}{
			"ids": []string{
				"${data.container.id}",
			},
		},
	}
	cfg, _ := common.NewConfigFrom(rawCfg)
    */
	return config{
		Key:    "logs",
        Host: "unix:///var/run/docker.sock", 
        DockerAPIVersion: "1.23",
	}
}
