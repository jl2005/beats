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
	"meitu.com.logs/topic":  "mytopic",

	"meitu.com.logs/format":  "json",
	"meitu.com.logs/format.field":  "message",

	"meitu.com.logs/format.type":  "regex",
	"meitu.com.logs/format.pattern":  "$regex",

    none|json|csv|nginx|apache2|regexp

    log-pilot 中使用access表示外挂目录


    // 按照容器名称修改的配置.指定名称的配置会覆盖上面的配置
	"meitu.com.logs.contain_name/include_lines": "^test5, ^test6",
	"meitu.com.logs.contain_name/exclude_lines": "^test7, ^test8",
	"meitu.com.logs.contain_name/topic": "mytopic",

    // 修改扩展目录的配置
    "meitu.com.logs.foobar/extern_paths": "/data/*.log",
    "meitu.com.logs.foobar/extern_path_conf.include_lines": "^test6, ^test7",
	"meitu.com.logs.foobar/extern_path_conf.exclude_lines": "^test8, ^test9",
	"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test10",
	"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",
	"meitu.com.logs.foobar/extern_path_conf.topic": "mytopic",

    // 修改扩展目录的配置
    "meitu.com.logs.foobar/extern_paths0": "/data/*.log",
    "meitu.com.logs.foobar/extern_path_conf0.include_lines": "^test6, ^test7",
	"meitu.com.logs.foobar/extern_path_conf0.exclude_lines": "^test8, ^test9",
	"meitu.com.logs.foobar/extern_path_conf0.multiline.pattern": "^test10",
	"meitu.com.logs.foobar/extern_path_conf0.multiline.negate": "true",
	"meitu.com.logs.foobar/extern_path_conf0.topic": "mytopic",
}
*/

type config struct {
	Prefix      string `config:"prefix"`
	Key         string `config:"key"`
	TopicPrefix string `config:"topic_prefix"`

	SkipContainers []string `config:"skip_containers"`

	Host             string `config:"host"`
	DockerAPIVersion string `config:"docker_api_version"`

	DefaultStdConfig    *common.Config `config:"stdconfig"`
	DefaultExpathConfig *common.Config `config:"expathconfig"`
}

func defaultConfig() config {
	conf := config{
		Prefix:      "meitu.com",
		Key:         "logs",
		TopicPrefix: "k8s",

		SkipContainers: []string{},

		Host:             "unix:///var/run/docker.sock",
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
		"fields_under_root": true,
	}
	conf.DefaultStdConfig, _ = common.NewConfigFrom(cfg)

	// 外挂目录的配置
	cfg = map[string]interface{}{
		"type":              "log",
		"paths":             "${data.extern_paths}",
		"fields_under_root": true,
	}
	conf.DefaultExpathConfig, _ = common.NewConfigFrom(cfg)
	return conf
}
