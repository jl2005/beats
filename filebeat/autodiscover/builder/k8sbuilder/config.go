package k8sbuilder

import (
	"github.com/elastic/beats/libbeat/common"
)

/*
"annotations": { // 修改docker标准输出的配置
	"meitu.com.logs/include_lines": "^test1, ^test2",
	"meitu.com.logs/exclude_lines": "^test3, ^test4",
	"meitu.com.logs/multiline.pattern": "^test5",
	"meitu.com.logs/multiline.negate":  "true",

    // 按照容器名称修改的配置.指定名称的配置会覆盖上面的配置
	"meitu.com.logs.contain_name/include_lines": "^test5, ^test6",
	"meitu.com.logs.contain_name/exclude_lines": "^test7, ^test8",

    // 修改扩展目录的配置 
    "meitu.com.logs.foobar/extern_paths": "/data/*.log", 
    "meitu.com.logs.foobar/extern_path_conf.include_lines": "^test6, ^test7",
	"meitu.com.logs.foobar/extern_path_conf.exclude_lines": "^test8, ^test9",
	"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test10",
	"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",

    // 修改扩展目录的配置 
    "meitu.com.logs.foobar/extern_paths0": "/data/*.log", 
    "meitu.com.logs.foobar/extern_path_conf0.include_lines": "^test6, ^test7",
	"meitu.com.logs.foobar/extern_path_conf0.exclude_lines": "^test8, ^test9",
	"meitu.com.logs.foobar/extern_path_conf0.multiline.pattern": "^test10",
	"meitu.com.logs.foobar/extern_path_conf0.multiline.negate": "true",
}
*/

func getDefaultStdConfig(id string) *common.Config {
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
    Prefix string `config:"prefix"`
	Key    string         `config:"key"`

	Host   string         `config:"host"`
	DockerAPIVersion string         `config:"docker_api_version"`

    DefaultStdConfig *common.Config `config:"stdconfig"`
    DefaultExpathConfig *common.Config `config:"expathconfig"`
}

func defaultConfig() config {
	conf := config{
        Prefix: "meitu.com",
		Key:    "logs",

        Host: "unix:///var/run/docker.sock", 
        DockerAPIVersion: "1.23",
	}

    // 标准输出的日志
	cfg := map[string]interface{}{
		"type": "docker",
		"containers": map[string]interface{}{
			"ids": []string{
                "${data.container.id}",
            },
		},
	}
	conf.DefaultStdConfig, _ = common.NewConfigFrom(cfg)

    // 外挂目录的配置
	cfg = map[string]interface{}{
		"type": "log",
		"paths": "${data.extern_paths}",
	}
	conf.DefaultExpathConfig, _ = common.NewConfigFrom(cfg)
    return conf
}
