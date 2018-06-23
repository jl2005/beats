package k8sbuilder

import (
    "context"
	"fmt"
    "path/filepath"

    "github.com/docker/docker/api/types"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
    "github.com/docker/docker/client"
)

const (
	Name = "k8sbuilder"
    Prefix = "meitu.com"

	multiline    = "multiline"
	includeLines = "include_lines"
	excludeLines = "exclude_lines"
)

func init() {
	autodiscover.Registry.AddBuilder("k8sbuilder", NewK8sBuilder)
}

type k8sBuilder struct {
    Key string

    client *client.Client

    conf config
}

func NewK8sBuilder(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("unable to unpack k8sbuidler config due to error: %v", err)
	}
    //TODO add http client
    c, err := client.NewClient(config.Host, config.DockerAPIVersion, nil, nil)
    if err != nil {
        return nil, err
    }
    k8sb := & k8sBuilder {
        conf: config,

        Key: config.Key,
        client: c,
    }

	return k8sb, nil
}

// Create config based on input k8s builder in the bus event
func (k8sb *k8sBuilder) CreateConfig(event bus.Event) []*common.Config {
	// 1. 标准输出的日志的配置
	mEvent := common.MapStr(event)

    id, err := mEvent.GetValue("kubernetes.container.id")
	if err != nil {
        logp.Err("get event container.id failed %s", err)
		return []*common.Config{}
	}

    hints := k8sb.getHints(mEvent)

	var configs []*common.Config
	if confs, err := k8sb.GetStdConfigs(id.(string), mEvent, hints); err != nil {
		logp.Err("create stdout/stderr config failed. %s", err)
	} else {
		configs = append(configs, confs...)
	}
	// 2. 容器内文件目录的日志
	if confs, err := k8sb.GetExPathConfig(id.(string), mEvent, hints); err != nil {
		logp.Err("create extern path config failed. %s", err)
	} else {
		configs = append(configs, confs...)
	}

	return configs
}

func (k8sb *k8sBuilder) getHints(event common.MapStr) common.MapStr {
    hints := common.MapStr{}
	name, err := event.GetValue("kubernetes.container.name")
	if err != nil {
        logp.Err("NOT found container.name for event %s", event.String())
        return hints
	}

	annInterface, err := event.GetValue("kubernetes.annotations")
	if err != nil {
        logp.Warn("get kubernetes.annotations failed %s", err)
        return hints
	}

	ann, ok  := annInterface.(common.MapStr)
    if !ok {
        return hints
    }
    logp.Debug(Name, "get annotations %s", ann.String())

    hints = builder.GenerateHints(ann, name.(string), Prefix)
    logp.Debug(Name, "%s get hints %s",name.(string), hints.String())
    return hints
}

func (k8sb *k8sBuilder) GetStdConfigs(id string, event, hints common.MapStr) ([]*common.Config, error) {
		// 获取docker的默认配置
	config := getDefaultDockerConfig(id)

	// 获取处理配置
    processConfig := getProcessConfig(k8sb.Key, hints)

	if err := config.Merge(processConfig); err != nil {
		logp.Err("merge process config failed %s", err)
	}

	logp.Debug(Name, "%s stdout config %s", id, common.ConfigDebugString(config, false))
	return []*common.Config{config}, nil
}


func (k8sb *k8sBuilder) GetExPathConfig(id string, event, hints common.MapStr) ([]*common.Config, error) {
	// 增加额外路径解析
    var configs []*common.Config
    path := builder.GetHintString(hints, k8sb.Key, "extern_path")
    if len(path) == 0 {
        return configs, nil
    }
    //TODO 支持多个路径
	absPath := k8sb.getAbsPath(id, path)
    if len(absPath) == 0 {
        logp.Err("get abs path failed. id=%s path=%s", id, path)
        return configs, nil
    }
	config := k8sb.genConfig(hints, absPath)
	configs = append(configs, config...)
    return configs, nil
}

func (k8sb *k8sBuilder) getAbsPath(id, path string) string {
    //TODO connect docker get path 
    containerJSON, err := k8sb.client.ContainerInspect(context.Background(), id)
    if err != nil {
        logp.Err("get container info failed %s", err)
        return ""
    }
    mountsMap := make(map[string]types.MountPoint)
    for _, mount := range containerJSON.Mounts {
        mountsMap[mount.Destination] = mount
    }
    return hostDirOf(path, mountsMap)
}

func hostDirOf(path string, mounts map[string]types.MountPoint) string {
    confPath := path
    for {
        if point, ok := mounts[path]; ok {
            if confPath == path {
                return point.Source
            } else {
                relPath, err := filepath.Rel(path, confPath)
                if err != nil {
                    panic(err)
                }
                return fmt.Sprintf("%s/%s", point.Source, relPath)
            }
        }
        path = filepath.Dir(path)
        if path == "/" || path == "." {
            break
        }
    }
    return ""
}

func (k8sb *k8sBuilder) genConfig(hints common.MapStr, path string) []*common.Config {
    config := getDefaultExternConfig(path)

	// 获取处理配置
    processConfig := getProcessConfig(k8sb.Key+".extern_path_conf", hints)
    logp.Debug(Name, "hints %s extern_path_conf %s", hints.String(), processConfig.String())

	if err := config.Merge(processConfig); err != nil {
		logp.Err("merge process config failed %s, config=%s, processConfig=%s", err, common.ConfigDebugString(config, false), processConfig.String())
	}

	logp.Debug(Name, "extern config %s", common.ConfigDebugString(config, false))
	return []*common.Config{config}
}

func getProcessConfig(prefix string, hints common.MapStr) common.MapStr {
    processConfig := common.MapStr{}
	mline := builder.GetHintMapStr(hints, prefix, multiline)
	if len(mline) != 0 {
		processConfig.Put(multiline, mline)
	}
	if ilines := builder.GetHintAsList(hints, prefix, includeLines); len(ilines) != 0 {
		processConfig.Put(includeLines, ilines)
	}
	if elines := builder.GetHintAsList(hints, prefix, excludeLines); len(elines) != 0 {
		processConfig.Put(excludeLines, elines)
	}

	logp.Debug(Name, "process config is '%s', prefix=%s hints=%s", processConfig.String(), prefix, hints.String())
    return processConfig
}

