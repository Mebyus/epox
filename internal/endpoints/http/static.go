package http

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

func (p *provider) Index(c *hits.Context) error {
	session, err := p.tryLoadSession(c)
	if err != nil {
		return err
	}

	buf := p.getRenBuf()
	defer p.putRenBuf(buf)

	modtime, err := renderIndexPage(buf, p.index, session)
	if err != nil {
		return err
	}

	content := hits.Content{
		Type:    "text/html; charset=utf-8",
		ModTime: modtime,
		Data:    buf,
		Size:    buf.Size(),
	}
	return c.RenderContent(&content)
}

func (p *provider) Static(c *hits.Context) error {
	_, path, ok := strings.Cut(c.Req.URL.Path, "static/")
	if !ok || path == "" {
		return hits.ErrNotFound
	}
	if strings.Contains(path, "..") {
		return errors.New("bad url")
	}

	fin, err := c.RenderFile(filepath.Join(p.static, path))
	if !fin {
		return err
	}
	return nil
}

const indexPrelude = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta name="description" content="Personal task manager" />
  <title>Epox</title>
  <link rel="icon" href="data:," />
  <link rel="stylesheet" href="/ext/static/reset.css" />
  <link rel="stylesheet" href="/ext/static/styles.css" />
  <script src="/ext/static/main.mjs" type="module"></script>
</head>
<body>
`

const indexEpilogue = `
</body>
</html>
`

// session can be nil in which case unathorized page
// variant will be rendered
//
// Returns modtime of file specified by path.
func renderIndexPage(g *hits.RenBuf, path string, session *base.Session) (time.Time, error) {
	g.Puts(indexPrelude)

	if session != nil {
		g.Puts("<script>window.user = {id: \"")
		g.Puts(strconv.FormatUint(uint64(session.UserID), 10))
		g.Puts("\", session: {expires: \"")
		g.Puts(session.ExpireTime.Format(time.RFC3339))
		g.Puts("\"}}</script>\n")
	}

	_, modtime, err := hits.CopyFileBytes(g, path)
	if err != nil {
		return time.Time{}, err
	}

	g.Puts(indexEpilogue)

	return modtime, nil
}
