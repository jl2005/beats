package k8sbuilder

import (
    "os"
    "testing"
    "net/http"
    "reflect"
    "context"

	"github.com/stretchr/testify/assert"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/common"
    "github.com/bouk/monkey"
    "github.com/docker/docker/client"
    "github.com/docker/docker/api/types"
)

func TestMain(m *testing.M) {
    for _, arg := range os.Args {
        if arg == "-test.v=true" {
            logp.DevelopmentSetup()
            break
        }
    }
    m.Run()
}

func TestNewK8sBuilder(t *testing.T) {
	tests := []struct {
		msg    string
		event  bus.Event
		len    int
		result []common.MapStr
	}{
		{
			msg: "Empty event hints should return default config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: []common.MapStr{
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
				},
            },
		},
		{
			msg: "include|exclude_lines must be part of the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs/include_lines": "^test, ^test1",
						"meitu.com.logs/exclude_lines": "^test2, ^test3",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: []common.MapStr{ common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"include_lines": []interface{}{"^test", "^test1"},
				"exclude_lines": []interface{}{"^test2", "^test3"},
			}},
		},
		{
			msg: "contain config replace include|exclude_lines must be part of the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs/include_lines": "^test1, ^test2",
						"meitu.com.logs/exclude_lines": "^test3, ^test4",
						"meitu.com.logs.foobar/include_lines": "^test5, ^test6",
						"meitu.com.logs.foobar/exclude_lines": "^test7, ^test8",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: []common.MapStr{ common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"include_lines": []interface{}{"^test5", "^test6"},
				"exclude_lines": []interface{}{"^test7", "^test8"},
			}},
		},
		{
			msg: "multiline config must have a multiline in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs/multiline.pattern": "^test",
						"meitu.com.logs/multiline.negate":  "true",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: []common.MapStr{ common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"multiline": map[string]interface{}{
					"pattern": "^test",
					"negate":  "true",
				},
			}},
		},
		{
			msg: "extern path include|exclude_lines must be part of the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs.foobar/extern_paths": "/data/*.log",
						"meitu.com.logs.foobar/extern_path_conf.include_lines": "^test, ^test1",
						"meitu.com.logs.foobar/extern_path_conf.exclude_lines": "^test2, ^test3",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 2,
			result: []common.MapStr{ 
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{}{"/node/path/data/*.log"},
					"include_lines": []interface{}{"^test", "^test1"},
					"exclude_lines": []interface{}{"^test2", "^test3"},
				},
            },
		},
		{
			msg: "extern path multiline config must have a multiline in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs.foobar/extern_paths": "/data/*.log",
						"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test",
						"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 2,
			result: []common.MapStr{ 
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{} {"/node/path/data/*.log"},
					"multiline": map[string]interface{}{
						"pattern": "^test",
						"negate":  "true",
					},
				},
            },
		},
		{
			msg: "mutil level extern path multiline config must have a multiline in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs.foobar/extern_paths": "/data/*/*.log",
						"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test",
						"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 2,
			result: []common.MapStr{ 
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{} {"/node/path/data/*/*.log"},
					"multiline": map[string]interface{}{
						"pattern": "^test",
						"negate":  "true",
					},
				},
            },
		},
		{
			msg: "mutil path for extern path multiline config must have a multiline in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs.foobar/extern_paths": "/data/*/*.log, /data1/*/*.log",
						"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test",
						"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 2,
			result: []common.MapStr{ 
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{} {
                        "/node/path/data/*/*.log",
                        "/node/path/data1/*/*.log",
                    },
					"multiline": map[string]interface{}{
						"pattern": "^test",
						"negate":  "true",
					},
				},
            },
		},
		{
			msg: "mutil extern path multiline config must have a multiline in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs.foobar/extern_paths0": "/data0/*.log",
						"meitu.com.logs.foobar/extern_path_conf0.multiline.pattern": "^test0",
						"meitu.com.logs.foobar/extern_path_conf0.multiline.negate": "true",
						"meitu.com.logs.foobar/extern_paths1": "/data1/*.log",
						"meitu.com.logs.foobar/extern_path_conf1.multiline.pattern": "^test1",
						"meitu.com.logs.foobar/extern_path_conf1.multiline.negate": "true",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 3,
			result: []common.MapStr{ 
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{} {"/node/path/data0/*.log"},
					"multiline": map[string]interface{}{
						"pattern": "^test0",
						"negate":  "true",
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{} {"/node/path/data1/*.log"},
					"multiline": map[string]interface{}{
						"pattern": "^test1",
						"negate":  "true",
					},
				},
            },
		},
		{
			msg: "all config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"annotations": testGetNestedAnnotations(common.MapStr{
						"meitu.com.logs/include_lines": "^test1, ^test2",
						"meitu.com.logs/exclude_lines": "^test3, ^test4",
						"meitu.com.logs/multiline.pattern": "^test5",
						"meitu.com.logs/multiline.negate":  "true",
						"meitu.com.logs.foobar/extern_paths": "/data/*.log",
						"meitu.com.logs.foobar/extern_path_conf.include_lines": "^test6, ^test7",
						"meitu.com.logs.foobar/extern_path_conf.exclude_lines": "^test8, ^test9",
						"meitu.com.logs.foobar/extern_path_conf.multiline.pattern": "^test10",
						"meitu.com.logs.foobar/extern_path_conf.multiline.negate": "true",
					}),
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 2,
			result: []common.MapStr{ 
                common.MapStr{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"include_lines": []interface{}{"^test1", "^test2"},
					"exclude_lines": []interface{}{"^test3", "^test4"},
					"multiline": map[string]interface{}{
						"pattern": "^test5",
						"negate":  "true",
					},
				},
                common.MapStr{
					"type": "log",
                    "paths": []interface{} {"/node/path/data/*.log"},
					"include_lines": []interface{}{"^test6", "^test7"},
					"exclude_lines": []interface{}{"^test8", "^test9"},
					"multiline": map[string]interface{}{
						"pattern": "^test10",
						"negate":  "true",
					},
				},
            },
		},
	}

    // mock docker client
    var cli = &client.Client{}
    monkey.Patch(client.NewClient, func(host string, version string, client *http.Client, httpHeaders map[string]string) (*client.Client, error){
        return cli, nil
    })
    monkey.PatchInstanceMethod(reflect.TypeOf(cli), "ContainerInspect", func(_ *client.Client, ctx context.Context, containerID string) (types.ContainerJSON, error) {
        js := types.ContainerJSON{
            Mounts: []types.MountPoint {
                types.MountPoint {
                    Source: "/node/path/data",
                    Destination: "/data",
                },
                types.MountPoint {
                    Source: "/node/path/data0",
                    Destination: "/data0",
                },
                types.MountPoint {
                    Source: "/node/path/data1",
                    Destination: "/data1",
                },
            },
        }
        return js, nil
    })

	for _, test := range tests {
		cfg, _ := common.NewConfigFrom(map[string]interface{}{
			"key": "logs",
		})

		b, err := NewK8sBuilder(cfg)
		if err != nil {
			t.Fatal(err)
		}

		cfgs := b.CreateConfig(test.event)
		assert.Equal(t, test.len, len(cfgs), test.msg)

        for i := range cfgs {
			config := common.MapStr{}
			err := cfgs[i].Unpack(&config)
			assert.Nil(t, err, test.msg)
			assert.Equal(t, test.result[i], config, test.msg)
        }
	}
}

func testGetNestedAnnotations(in common.MapStr) common.MapStr {
    out := common.MapStr{}

    for k, v := range in {
        out.Put(k, v)
    }
    return out
}
