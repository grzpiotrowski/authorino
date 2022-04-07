package evaluators

import (
	"context"
	"fmt"

	"github.com/kuadrant/authorino/pkg/auth"
	"github.com/kuadrant/authorino/pkg/evaluators/metadata"
	"github.com/kuadrant/authorino/pkg/json"
	"github.com/kuadrant/authorino/pkg/log"
)

const (
	metadataUserInfo    = "METADATA_USERINFO"
	metadataUMA         = "METADATA_UMA"
	metadataGenericHTTP = "METADATA_GENERIC_HTTP"
)

type MetadataConfig struct {
	Name       string                         `yaml:"name"`
	Priority   int                            `yaml:"priority"`
	Conditions []json.JSONPatternMatchingRule `yaml:"conditions"`
	Metrics    bool                           `yaml:"metrics"`

	UserInfo    *metadata.UserInfo    `yaml:"userinfo,omitempty"`
	UMA         *metadata.UMA         `yaml:"uma,omitempty"`
	GenericHTTP *metadata.GenericHttp `yaml:"http,omitempty"`
}

func (config *MetadataConfig) GetAuthConfigEvaluator() auth.AuthConfigEvaluator {
	switch config.GetType() {
	case metadataUserInfo:
		return config.UserInfo
	case metadataUMA:
		return config.UMA
	case metadataGenericHTTP:
		return config.GenericHTTP
	default:
		return nil
	}
}

// impl:AuthConfigEvaluator

func (config *MetadataConfig) Call(pipeline auth.AuthPipeline, ctx context.Context) (interface{}, error) {
	if evaluator := config.GetAuthConfigEvaluator(); evaluator != nil {
		logger := log.FromContext(ctx).WithName("metadata")
		return evaluator.Call(pipeline, log.IntoContext(ctx, logger))
	} else {
		return nil, fmt.Errorf("invalid metadata config")
	}
}

// impl:NamedEvaluator

func (config *MetadataConfig) GetName() string {
	return config.Name
}

// impl:TypedEvaluator

func (config *MetadataConfig) GetType() string {
	switch {
	case config.UserInfo != nil:
		return metadataUserInfo
	case config.UMA != nil:
		return metadataUMA
	case config.GenericHTTP != nil:
		return metadataGenericHTTP
	default:
		return ""
	}
}

// impl:Prioritizable

func (config *MetadataConfig) GetPriority() int {
	return config.Priority
}

// impl:ConditionalEvaluator

func (config *MetadataConfig) GetConditions() []json.JSONPatternMatchingRule {
	return config.Conditions
}

// impl:metrics.Object

func (config *MetadataConfig) MetricsEnabled() bool {
	return config.Metrics
}