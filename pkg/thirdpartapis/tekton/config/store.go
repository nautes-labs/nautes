/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"context"

	"knative.dev/pkg/configmap"
)

type cfgKey struct{}

// Config holds the collection of configurations that we attach to contexts.

type Config struct {
	Defaults       *Defaults
	FeatureFlags   *FeatureFlags
	ArtifactBucket *ArtifactBucket
	ArtifactPVC    *ArtifactPVC
	Metrics        *Metrics
}

// FromContext extracts a Config from the provided context.
func FromContext(ctx context.Context) *Config {
	x, ok := ctx.Value(cfgKey{}).(*Config)
	if ok {
		return x
	}
	return nil
}

// FromContextOrDefaults is like FromContext, but when no Config is attached it
// returns a Config populated with the defaults for each of the Config fields.
func FromContextOrDefaults(ctx context.Context) *Config {
	if cfg := FromContext(ctx); cfg != nil {
		return cfg
	}
	defaults, _ := NewDefaultsFromMap(map[string]string{})
	featureFlags, _ := NewFeatureFlagsFromMap(map[string]string{})
	artifactBucket, _ := NewArtifactBucketFromMap(map[string]string{})
	artifactPVC, _ := NewArtifactPVCFromMap(map[string]string{})
	metrics, _ := newMetricsFromMap(map[string]string{})
	return &Config{
		Defaults:       defaults,
		FeatureFlags:   featureFlags,
		ArtifactBucket: artifactBucket,
		ArtifactPVC:    artifactPVC,
		Metrics:        metrics,
	}
}

// ToContext attaches the provided Config to the provided context, returning the
// new context with the Config attached.
func ToContext(ctx context.Context, c *Config) context.Context {
	return context.WithValue(ctx, cfgKey{}, c)
}

// Store is a typed wrapper around configmap.Untyped store to handle our configmaps.

type Store struct {
	*configmap.UntypedStore
}

// NewStore creates a new store of Configs and optionally calls functions when ConfigMaps are updated.
func NewStore(logger configmap.Logger, onAfterStore ...func(name string, value interface{})) *Store {
	store := &Store{
		UntypedStore: configmap.NewUntypedStore(
			"defaults/features/artifacts",
			logger,
			configmap.Constructors{
				GetDefaultsConfigName():       NewDefaultsFromConfigMap,
				GetFeatureFlagsConfigName():   NewFeatureFlagsFromConfigMap,
				GetArtifactBucketConfigName(): NewArtifactBucketFromConfigMap,
				GetArtifactPVCConfigName():    NewArtifactPVCFromConfigMap,
				GetMetricsConfigName():        NewMetricsFromConfigMap,
			},
			onAfterStore...,
		),
	}

	return store
}

// ToContext attaches the current Config state to the provided context.
func (s *Store) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, s.Load())
}

// Load creates a Config from the current config state of the Store.
func (s *Store) Load() *Config {
	defaults := s.UntypedLoad(GetDefaultsConfigName())
	if defaults == nil {
		defaults, _ = NewDefaultsFromMap(map[string]string{})
	}
	featureFlags := s.UntypedLoad(GetFeatureFlagsConfigName())
	if featureFlags == nil {
		featureFlags, _ = NewFeatureFlagsFromMap(map[string]string{})
	}
	artifactBucket := s.UntypedLoad(GetArtifactBucketConfigName())
	if artifactBucket == nil {
		artifactBucket, _ = NewArtifactBucketFromMap(map[string]string{})
	}
	artifactPVC := s.UntypedLoad(GetArtifactPVCConfigName())
	if artifactPVC == nil {
		artifactPVC, _ = NewArtifactPVCFromMap(map[string]string{})
	}

	metrics := s.UntypedLoad(GetMetricsConfigName())
	if metrics == nil {
		metrics, _ = newMetricsFromMap(map[string]string{})
	}
	return &Config{
		Defaults:       defaults.(*Defaults).DeepCopy(),
		FeatureFlags:   featureFlags.(*FeatureFlags).DeepCopy(),
		ArtifactBucket: artifactBucket.(*ArtifactBucket).DeepCopy(),
		ArtifactPVC:    artifactPVC.(*ArtifactPVC).DeepCopy(),
		Metrics:        metrics.(*Metrics).DeepCopy(),
	}
}
