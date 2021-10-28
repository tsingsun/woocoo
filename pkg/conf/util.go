package conf

import (
	"net/url"
	"path/filepath"
)

// ConvertRefFileDSN if dsn use relative path,covert to abs path by a specified dir(use app run dir usually)
func ConvertRefFileDSN(base, dsn string) (*url.URL, error) {
	uri, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	if uri.Opaque != "" {
		if !filepath.IsAbs(uri.Opaque) {
			uri.Opaque = filepath.Join(base, uri.Opaque)
		}
	}
	if uri.Path != "" {
		if !filepath.IsAbs(uri.Path) {
			uri.Path = filepath.Join(base, uri.Path)
		}
	}
	return uri, nil
}

// GetPathFromFileUrl get path of file dsn
func GetPathFromFileUrl(fileUrl *url.URL) string {
	if fileUrl.Path != "" {
		return fileUrl.Path
	}
	if fileUrl.Opaque != "" {
		return fileUrl.Opaque
	}
	return ""
}
