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
	TemplateProviderCount  int                  `json:"templateProviderCount"`
}

type TargetResult struct {
	Path                   string               `json:"path"`
	DetectedProviders      []string             `json:"detectedProviders,omitempty"`
	FinalProviders         []string             `json:"finalProviders"`
	DetectionResults       []provider.Result    `json:"detectionResults,omitempty"`
	RemoteProviderWarnings []string             `json:"-"`
	FileAction             gitignore.FileAction `json:"fileAction"`
	TemplateProviderCount  int                  `json:"templateProviderCount"`
}
