package autodiscover

import (
	// include all filebeat specific builders
	_ "github.com/elastic/beats/filebeat/autodiscover/builder/hints"
	_ "github.com/elastic/beats/filebeat/autodiscover/builder/k8sbuilder"
)
