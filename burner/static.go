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
	if s.cfg.Developing {
		return http.FS(os.DirFS("./burner/static"))
	}

	subFs, err := fs.Sub(staticFiles, "prodStatic")
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
