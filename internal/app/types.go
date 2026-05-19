package app

import (
	"github.com/aaronflorey/genignore/internal/gitignore"
	"github.com/aaronflorey/genignore/internal/provider"
)

type CommandResult struct {
	Command                string               `json:"command"`
	CWD                    string               `json:"cwd"`
	Targets                []TargetResult       `json:"targets,omitempty"`
	DetectedProviders      []string             `json:"detectedProviders,omitempty"`
	IncludedProviders      []string             `json:"includedProviders,omitempty"`
	ExcludedProviders      []string             `json:"excludedProviders,omitempty"`
	AddedProviders         []string             `json:"addedProviders,omitempty"`
	FinalProviders         []string             `json:"finalProviders"`
	UnsupportedKeyWarnings []string             `json:"unsupportedKeyWarnings,omitempty"`
	RuntimeWarnings        []string             `json:"runtimeWarnings,omitempty"`
	RemoteProviderWarnings []string             `json:"remoteProviderWarnings,omitempty"`
	DetectionResults       []provider.Result    `json:"detectionResults,omitempty"`
	FileAction             gitignore.FileAction `json:"fileAction"`
	PreviewOnly            bool                 `json:"previewOnly,omitempty"`
	Diff                   string               `json:"diff,omitempty"`
	TemplateProviderCount  int                  `json:"templateProviderCount"`
}

type ResolveOptions struct {
	Include []string
	Exclude []string
	Verbose bool
}

type ResolveResult struct {
	Command                string            `json:"command"`
	CWD                    string            `json:"cwd"`
	DetectedProviders      []string          `json:"detectedProviders,omitempty"`
	IncludedProviders      []string          `json:"includedProviders,omitempty"`
	ExcludedProviders      []string          `json:"excludedProviders,omitempty"`
	FinalProviders         []string          `json:"finalProviders"`
	UnsupportedKeyWarnings []string          `json:"unsupportedKeyWarnings,omitempty"`
	DetectionResults       []provider.Result `json:"detectionResults,omitempty"`
}

type TargetResult struct {
	Path                   string               `json:"path"`
	DetectedProviders      []string             `json:"detectedProviders,omitempty"`
	FinalProviders         []string             `json:"finalProviders"`
	DetectionResults       []provider.Result    `json:"detectionResults,omitempty"`
	RemoteProviderWarnings []string             `json:"-"`
	FileAction             gitignore.FileAction `json:"fileAction"`
	Diff                   string               `json:"-"`
	TemplateProviderCount  int                  `json:"templateProviderCount"`
}

type DoctorOptions struct {
	Include []string
	Exclude []string
}

type DoctorResult struct {
	Command                string            `json:"command"`
	CWD                    string            `json:"cwd"`
	DetectedProviders      []string          `json:"detectedProviders,omitempty"`
	IncludedProviders      []string          `json:"includedProviders,omitempty"`
	ExcludedProviders      []string          `json:"excludedProviders,omitempty"`
	FinalProviders         []string          `json:"finalProviders"`
	UnsupportedKeyWarnings []string          `json:"unsupportedKeyWarnings,omitempty"`
	RuntimeWarnings        []string          `json:"runtimeWarnings,omitempty"`
	RemoteProviderWarnings []string          `json:"remoteProviderWarnings,omitempty"`
	Detections             []DoctorDetection `json:"detections,omitempty"`
	Runtime                DoctorRuntime     `json:"runtime"`
	Provenance             []string          `json:"provenance,omitempty"`
}

type DoctorDetection struct {
	Key      string `json:"key"`
	Matched  bool   `json:"matched"`
	Origin   string `json:"origin"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence,omitempty"`
	Error    string `json:"error,omitempty"`
}

type DoctorRuntime struct {
	UpstreamCommit    string             `json:"upstreamCommit"`
	Offline           bool               `json:"offline"`
	RemoteProviders   []string           `json:"remoteProviders,omitempty"`
	EmbeddedProviders []string           `json:"embeddedProviders,omitempty"`
	CacheEntries      []DoctorCacheEntry `json:"cacheEntries,omitempty"`
	Decisions         []string           `json:"decisions,omitempty"`
}

type DoctorCacheEntry struct {
	Provider string `json:"provider"`
	State    string `json:"state"`
	Detail   string `json:"detail,omitempty"`
}
