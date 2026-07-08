package providercatalog

import "github.com/aaronflorey/genignore/internal/templatecatalog"

func InitError() error {
	return templatecatalog.InitError()
}

func RemoteSupportedKeys() []string {
	return templatecatalog.Providers()
}
