package k8sbuilder

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	Name = "k8sbuilder"

	multiline    = "multiline"
	includeLines = "include_lines"
	excludeLines = "exclude_lines"
	topic        = "topic"
	format       = "format"

	externPaths    = "extern_paths"
	externPathConf = "extern_path_conf"

	MaxExpathConfig = 5
)

func init() {
	autodiscover.Registry.AddBuilder("k8sbuilder", NewK8sBuilder)
}

type k8sBuilder struct {
	Prefix string
	Key    string

	dockerClient *client.Client
	conf         config
}

func NewK8sBuilder(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("unable to unpack k8sbuidler config due to error: %v", err)
	}
	// connect docker for get extern path
	cli, err := client.NewClient(config.Host, config.DockerAPIVersion, nil, nil)
	if err != nil {
		return nil, err
	}
	k8sb := &k8sBuilder{
		Prefix:       config.Prefix,
		Key:          config.Key,
		dockerClient: cli,

		conf: config,
	}

	return k8sb, nil
}

// Create config based on input k8s builder in the bus event
func (k8sb *k8sBuilder) CreateConfig(event bus.Event) []*common.Config {
	mEvent := common.MapStr(event)

	hints := k8sb.getHints(mEvent)

	var configs []*common.Config
	// 1. 标准输出的日志的配置
	if confs, err := k8sb.getStdConfigs(mEvent, hints); err != nil {
		logp.Err("create stdout/stderr config failed. %s", err)
	} else {
		configs = append(configs, confs...)
	}
	// 2. 容器内文件目录的日志
	if confs, err := k8sb.getExpathConfig(mEvent, hints); err != nil {
		logp.Err("create extern path config failed. %s", err)
	} else {
		configs = append(configs, confs...)
	}

	return configs
}

func (k8sb *k8sBuilder) getHints(event common.MapStr) (hints common.MapStr) {
	name, err := event.GetValue("kubernetes.container.name")
	if err != nil {
		logp.Err("NOT found kubernetes.container.name from event %s", event.String())
		return hints
	}

	annInterface, err := event.GetValue("kubernetes.annotations")
	if err != nil {
		logp.Warn("get kubernetes.annotations failed %s", err)
		return hints
	}

	ann, ok := annInterface.(common.MapStr)
	if !ok {
		logp.Err("kubernetes annotations is not common.MapStr type %s", event.String())
		return hints
	}
	hints = builder.GenerateHints(ann, name.(string), k8sb.Prefix)
	logp.Debug(Name, "get hints '%s' for container '%s', annotations is %s", hints.String(), name.(string), ann.String())
	return hints
}

func (k8sb *k8sBuilder) getStdConfigs(event, hints common.MapStr) ([]*common.Config, error) {
	// 获取docker的默认配置
	config, err := common.NewConfigFrom(k8sb.conf.DefaultStdConfig)
	if err != nil {
		return nil, fmt.Errorf("get default std config failed. %s", err)
	}

	// 获取处理配置
	processConfig := getProcessConfig(k8sb.Key, hints, k8sb.conf.TopicPrefix, "stdlog")

	if err := config.Merge(processConfig); err != nil {
		logp.Err("merge process config failed %s", err)
	}

	logp.Debug(Name, "stdout config %s", common.ConfigDebugString(config, false))
	configs := template.ApplyConfigTemplate(bus.Event(event), []*common.Config{config})
	for i := range configs {
		logp.Debug(Name, "alfter apply template stdout config %s", common.ConfigDebugString(configs[i], false))
	}
	return configs, nil
}

func (k8sb *k8sBuilder) getExpathConfig(event, hints common.MapStr) ([]*common.Config, error) {
	var ret []*common.Config

	// 获取默认不增加index的配置
	externPathKey := fmt.Sprintf("%s", externPaths)
	externPathConfKey := fmt.Sprintf("%s.%s", k8sb.Key, externPathConf)
	configs, err := k8sb.getExpathConfigForIndex(event, hints, externPathKey, externPathConfKey)
	if err != nil {
		return configs, err
	}
	if len(configs) > 0 {
		ret = append(ret, configs...)
	}

	for i := 0; i < MaxExpathConfig; i++ {
		externPathKey := fmt.Sprintf("%s%d", externPaths, i)
		externPathConfKey := fmt.Sprintf("%s.%s%d", k8sb.Key, externPathConf, i)
		configs, err := k8sb.getExpathConfigForIndex(event, hints, externPathKey, externPathConfKey)
		if err != nil {
			return configs, err
		}
		if len(configs) == 0 {
			break
		}
		ret = append(ret, configs...)
	}
	return ret, nil
}

func (k8sb *k8sBuilder) getExpathConfigForIndex(event, hints common.MapStr, externPathKey, externPathConfKey string) ([]*common.Config, error) {
	var configs []*common.Config
	paths := builder.GetHintAsList(hints, k8sb.Key, externPathKey)
	if len(paths) == 0 {
		return configs, nil
	}

	absPaths := k8sb.getAbsPaths(paths, event)
	if len(absPaths) == 0 {
		return configs, fmt.Errorf("can't get abstract path for %v", paths)
	}

	config, err := common.NewConfigFrom(k8sb.conf.DefaultExpathConfig)
	if err != nil {
		return configs, fmt.Errorf("get default std config failed. %s", err)
	}

	// 获取处理配置
	processConfig := getProcessConfig(externPathConfKey, hints, k8sb.conf.TopicPrefix, externPathKey)
	logp.Debug(Name, "hints %s extern_path_conf %s", hints.String(), processConfig.String())

	if err := config.Merge(processConfig); err != nil {
		logp.Err("merge process config failed %s, config=%s, processConfig=%s", err, common.ConfigDebugString(config, false), processConfig.String())
	}

	logp.Debug(Name, "extern path config %s", common.ConfigDebugString(config, false))
	event.Put("extern_paths", absPaths)
	configs = template.ApplyConfigTemplate(bus.Event(event), []*common.Config{config})
	event.Delete("extern_paths")
	for i := range configs {
		logp.Debug(Name, "alfter apply template extern path config %s", common.ConfigDebugString(configs[i], false))
	}
	return configs, nil
}

func (k8sb *k8sBuilder) getAbsPaths(paths []string, event common.MapStr) []string {
	if len(paths) == 0 {
		return nil
	}

	id, err := event.GetValue("kubernetes.container.id")
	if err != nil {
		logp.Err("get kubernetes.container.id failed %s", err)
		return nil
	}

	// TODO 需要使用一个超时时间的context
	containerJSON, err := k8sb.dockerClient.ContainerInspect(context.Background(), id.(string))
	if err != nil {
		logp.Err("get container info failed %s", err)
		return nil
	}
	mountsMap := make(map[string]string)
	for _, mount := range containerJSON.Mounts {
		mountsMap[mount.Destination] = mount.Source
	}
	ret := make([]string, len(paths))
	for i := range paths {
		ret[i] = hostDirOf(paths[i], mountsMap)
	}
	return ret
}

//FIXME 这个应该不能支持 /*/*/*.log 这种形式的拼接
func hostDirOf(path string, mounts map[string]string) string {
	confPath := path
	for {
		if source, ok := mounts[path]; ok {
			if confPath == path {
				return source
			} else {
				relPath, err := filepath.Rel(path, confPath)
				if err != nil {
					logp.Err("filepath.Rel('%s', '%s'). %s", path, confPath, err)
					return ""
				}
				return fmt.Sprintf("%s/%s", source, relPath)
			}
		}
		path = filepath.Dir(path)
		if path == "/" || path == "." {
			break
		}
	}
	return ""
}

func getProcessConfig(key string, hints common.MapStr, topicPrefix, logType string) common.MapStr {
	processConfig := common.MapStr{}
	if mline := builder.GetHintMapStr(hints, key, multiline); len(mline) != 0 {
		processConfig.Put(multiline, mline)
	}
	if ilines := builder.GetHintAsList(hints, key, includeLines); len(ilines) != 0 {
		processConfig.Put(includeLines, ilines)
	}
	if elines := builder.GetHintAsList(hints, key, excludeLines); len(elines) != 0 {
		processConfig.Put(excludeLines, elines)
	}
	//TODO 这个放在这里是否合适
	if t := builder.GetHintString(hints, key, topic); len(t) != 0 {
		processConfig.Put("fields.topic", topicPrefix+"_${data.container.namespaces}_${data.container.name}_"+t)
	} else {
		processConfig.Put("fields.topic", topicPrefix+"_${data.container.namespaces}_${data.container.name}_"+logType)
	}

	if params := builder.GetHintMapStr(hints, key, format); len(params) != 0 {
		processConfig.Put("processors", []common.MapStr{common.MapStr{"mydecode_field": params}})
	} else {
		processConfig.Put("processors", []common.MapStr{common.MapStr{"mydecode_field": common.MapStr{"type": "json"}}})
	}

	logp.Debug(Name, "key '%s' process config is '%s', hints '%s'", key, processConfig.String(), hints.String())
	return processConfig
}
