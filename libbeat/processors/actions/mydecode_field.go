package actions

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/pkg/errors"
)

type myDecodeField struct {
	config *myDecodeFieldConfig

	unmatchTarget string

	re *regexp.Regexp
}

type myDecodeFieldConfig struct {
	Type             string `config:"type"`
	Field            string `config:"field"`
	PreserveOldField bool   `config:"preserve_old_field"`
	Target           string `config:"target"`

	// json 相关的配置
	MaxDepth      int  `config:"max_depth" validate:"min=1"`
	ProcessArray  bool `config:"process_array"`
	OverwriteKeys bool `config:"overwrite_keys"`

	//regex 相关的配置
	Expression string `config:"expression"`
	//TODO 增加时间字段的解析,其实可以不加
}

var (
	// 默认解析json，将解析结果放到log字段下
	defaultmyDecodeFieldConfig = &myDecodeFieldConfig{
		Type:             "json",
		Field:            "message",
		PreserveOldField: false,
		Target:           "log",

		MaxDepth:     10,
		ProcessArray: true,
	}
)

func init() {
	processors.RegisterPlugin("mydecode_field",
		configChecked(newMyDecodeField,
			requireFields("type"),
			allowedFields("type", "field", "preserve_old_field", "target", "max_depth", "overwrite_keys", "process_array", "expression", "when")))
}

func newMyDecodeField(c *common.Config) (processors.Processor, error) {
	config := defaultmyDecodeFieldConfig
	err := c.Unpack(config)
	if err != nil {
		logp.Warn("Error unpacking config for mydecode_field")
		return nil, fmt.Errorf("fail to unpack the mydecode_field configuration: %s", err)
	}

	decode := &myDecodeField{
		config: config,
	}
	if err := decode.checkConfig(); err != nil {
		return nil, err
	}

	return decode, nil
}

func (f *myDecodeField) checkConfig() error {
	if f.config == nil {
		return fmt.Errorf("NOT set config")
	}
	switch f.config.Type {
	case "json":
		if f.config.MaxDepth <= 0 {
			return fmt.Errorf("mydecode_field maxdepth<=0")
		}
		f.unmatchTarget = f.config.Target + ".nonjson"
	case "regex":
		if len(f.config.Expression) > 0 {
			re, err := regexp.Compile(f.config.Expression)
			if err != nil {
				return fmt.Errorf("mydecode_field compile '%s' failed. %s", f.config.Expression, err)
			}
			f.re = re
		}
		f.unmatchTarget = f.config.Target + ".unmatch"
	default:
		return fmt.Errorf("mydecode_field unsupport type '%s'. current support json|regex", f.config.Type)
	}
	return nil
}

func (f *myDecodeField) Run(event *beat.Event) (*beat.Event, error) {
	switch f.config.Type {
	case "json":
		return f.jsonDecode(event)
	case "regex":
		return f.regexDecode(event)
	default:
		logp.Err("mydecode_field unsupport type '%s'. current support json|regex", f.config.Type)
	}
	return event, nil
}

func (f *myDecodeField) regexDecode(event *beat.Event) (*beat.Event, error) {
	if f.re == nil {
		logp.Err("not set regex")
		return event, nil
	}
	data, err := event.GetValue(f.config.Field)
	if err != nil {
		if errors.Cause(err) != common.ErrKeyNotFound {
			debug("Error trying to GetValue for field : %s in event : %v, err: %s", f.config.Field, event, err)
			return event, err
		}
		return event, nil
	}
	text, ok := data.(string)
	if !ok {
		// ignore non string fields when unmarshaling
		debug("field is not string: %s in event : %v, err: %s", f.config.Field, event, err)
		return event, nil
	}

	res := f.re.FindStringSubmatch(text)
	if len(res) != len(f.re.SubexpNames()) {
		if _, err = event.PutValue(f.unmatchTarget, text); err != nil {
			debug("Error trying to Put '%s' failed: event : %v, err: %s", f.unmatchTarget, event, err)
		}
		return event, nil
	}

	output := common.MapStr{}
	for i := 1; i < len(f.re.SubexpNames()); i++ {
		output.Put(f.re.SubexpNames()[i], res[i])
	}
	if !f.config.PreserveOldField {
		event.Delete(f.config.Field)
	}
	event.PutValue(f.config.Target, output)

	return event, nil
}

func (f *myDecodeField) jsonDecode(event *beat.Event) (*beat.Event, error) {
	data, err := event.GetValue(f.config.Field)
	if err != nil {
		if errors.Cause(err) != common.ErrKeyNotFound {
			debug("Error trying to GetValue for field : %s in event : %v, err: %s", f.config.Field, event, err)
			return event, err
		}
		return event, nil
	}

	text, ok := data.(string)
	if !ok {
		// ignore non string fields when unmarshaling
		debug("field is not string: %s in event : %v, err: %s", f.config.Field, event, err)
		return event, nil
	}

	var output interface{}
	err = unmarshal(f.config.MaxDepth, text, &output, f.config.ProcessArray)
	if err != nil {
		if _, err = event.PutValue(f.unmatchTarget, text); err != nil {
			debug("Error trying to Put '%s' failed: event : %v, err: %s", f.unmatchTarget, event, err)
		}
		return event, nil
	}

	if !f.config.PreserveOldField {
		event.Delete(f.config.Field)
	}
	if _, err = event.PutValue(f.config.Target, output); err != nil {
		debug("Error trying to PutValue log failed: event : %v, err: %s", event, err)
	}
	return event, nil
}

func (f myDecodeField) String() string {
	return "mydecode_field=" + f.config.Field
}
