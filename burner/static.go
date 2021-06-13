//go:generate
package burner

import (
	"embed"
	_ "embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

// CSS file name -- overridden by ldflags
var css = "styles.css"

//go:embed prodStatic
var staticFiles embed.FS

func (s *Server) getStaticFS() http.FileSystem {
	var static fs.FS
	var subDir string

	if s.cfg.Developing {
		static = os.DirFS("./static")
		subDir = "static"
	} else {
		static = staticFiles
		subDir = "prodStatic"
	}

	subFs, err := fs.Sub(static, subDir)
	if err != nil {
		log.WithField("dev", s.cfg.Developing).WithError(err).Fatal("failed to get sub fs")
		return nil
	}
	return http.FS(subFs)
}

type staticDetails struct {
	FontPath string
	CSS      string
	Logo     string
}

func (s *Server) getStaticDetails() staticDetails {
	return staticDetails{
		FontPath: s.cfg.StaticURL,
		CSS:      fmt.Sprintf("%s/%s", s.cfg.StaticURL, css),
		Logo:     fmt.Sprintf("%s/%s", s.cfg.StaticURL, "roger.svg"),
	}
}
