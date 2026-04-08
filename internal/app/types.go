package app

import (
	"github.com/aaronflorey/gitignore-gen/internal/gitignore"
	"github.com/aaronflorey/gitignore-gen/internal/provider"
)

type CommandResult struct {
	Command                string               `json:"command"`
	CWD                    string               `json:"cwd"`
	DetectedProviders      []string             `json:"detectedProviders,omitempty"`
	IncludedProviders      []string             `json:"includedProviders,omitempty"`
	ExcludedProviders      []string             `json:"excludedProviders,omitempty"`
	AddedProviders         []string             `json:"addedProviders,omitempty"`
	FinalProviders         []string             `json:"finalProviders"`
	UnsupportedKeyWarnings []string             `json:"unsupportedKeyWarnings,omitempty"`
	RemoteProviderWarnings []string             `json:"remoteProviderWarnings,omitempty"`
	DetectionResults       []provider.Result    `json:"detectionResults,omitempty"`
	FileAction             gitignore.FileAction `json:"fileAction"`
	TemplateProviderCount  int                  `json:"templateProviderCount"`
}
