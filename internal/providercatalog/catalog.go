package providercatalog

import "github.com/aaronflorey/genignore/internal/templatecatalog"

func InitError() error {
	return templatecatalog.InitError()
}

func SupportedKeys() []string {
	return templatecatalog.Providers()
}

func RemoteSupportedKeys() []string {
	return SupportedKeys()
}
