package main

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kataras/httpfs"
)

type (
	listPageData struct {
		Title   string // the document's title.
		Files   []fileInfoData
		RelPath string // the request path.
	}

	fileInfoData struct {
		Info     os.FileInfo
		ModTime  string // format-ed time.
		Path     string // the request path.
		RelPath  string // file path without the system directory itself (we are not exposing it to the user).
		Name     string // the html-escaped name.
		Download bool   // the file should be downloaded (attachment instead of inline view).
	}
)

func toBaseName(s string) string {
	n := len(s) - 1
	for i := n; i >= 0; i-- {
		if c := s[i]; c == '/' || c == '\\' {
			if i == n {
				// "s" ends with a slash, remove it and retry.
				return toBaseName(s[:n])
			}

			return s[i+1:] // return the rest, trimming the slash.
		}
	}

	return s
}

// DirListRich is a `DirListFunc` which can be passed to `Options.DirList` field
// to override the default file listing appearance.
// See `DirListRichTemplate` to modify the template, if necessary.
func DirListRich(options httpfs.DirListRichOptions) httpfs.DirListFunc {
	if options.Tmpl == nil {
		options.Tmpl = httpfs.DirListRichTemplate
	}

	return func(w http.ResponseWriter, r *http.Request, dirOptions httpfs.Options, dirName string, dir http.File) error {
		dirs, err := dir.Readdir(-1)
		if err != nil {
			return err
		}

		sortBy := r.URL.Query().Get("sort")
		switch sortBy {
		case "name":
			sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
		case "size":
			sort.Slice(dirs, func(i, j int) bool { return dirs[i].Size() < dirs[j].Size() })
		default:
			sort.Slice(dirs, func(i, j int) bool { return dirs[i].ModTime().After(dirs[j].ModTime()) })
		}

		title := options.Title
		if title == "" {
			title = fmt.Sprintf("List of %d files", len(dirs))
		}

		pageData := listPageData{
			Title:   title,
			Files:   make([]fileInfoData, 0, len(dirs)),
			RelPath: r.URL.Path,
		}

		for _, d := range dirs {
			name := toBaseName(d.Name())

			upath := path.Join(r.RequestURI, name)
			url := url.URL{Path: upath}

			viewName := name
			if d.IsDir() {
				viewName += "/"
			}

			shouldDownload := dirOptions.Attachments.Enable && !d.IsDir()
			pageData.Files = append(pageData.Files, fileInfoData{
				Info:     d,
				ModTime:  d.ModTime().UTC().Format(http.TimeFormat),
				Path:     url.String(),
				RelPath:  path.Join(r.URL.Path, name),
				Name:     html.EscapeString(viewName),
				Download: shouldDownload,
			})
		}

		return options.Tmpl.ExecuteTemplate(w, options.TmplName, pageData)
	}
}

var myHTMLTemplate = template.Must(template.New("dirlist.html").Funcs(template.FuncMap{
	"formatBytes": func(b int64) string {
		const unit = 1000
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
	},
	"formatTime": func(t time.Time) string {
		return t.Format("2006-01-02 15:04")
	},
	"isRoot": func(path string) bool {
		return path == "/" || path == ""
	},
	"split": func(s string, sep string) []string {
		return strings.Split(s, sep)
	},
}).Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
            font-size: 14px;  /* 调整基础字体大小 */
        }
        h1 {
            font-size: 18px;  /* 调整标题大小 */
        }
        th, td {
            font-size: 13px;  /* 调整表格字体大小 */
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
            background: white;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-top: 0;
            padding-bottom: 10px;
            border-bottom: 1px solid #eee;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        th {
            background-color: #f2f2f2;
            text-align: left;
            padding: 12px;
            font-weight: 500;
        }
        td {
            padding: 10px 12px;
            border-bottom: 1px solid #eee;
        }
        tr:hover {
            background-color: #f9f9f9;
        }
        a {
            color: #0066cc;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .dir-icon:before {
            content: "📁 ";
        }
        .file-icon:before {
            content: "📄 ";
        }
        .size {
            text-align: right;
            font-family: monospace;
        }
        .time {
            white-space: nowrap;
        }
        .breadcrumb {
            padding: 5px 15px;
            margin-bottom: 5px;
            background-color: #f5f5f5;
            border-radius: 4px;
        }
        .breadcrumb a {
            color: #0066cc;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
		<div class="breadcrumb">
			当前目录：
            {{$path := ""}}
            {{range $i, $part := split .RelPath "/"}}
                {{if $part}}
                    {{$path = printf "%s/%s" $path $part}}
                    {{if gt $i 0}} / {{end}}
                    <a href="{{$path}}">{{$part}}</a>
                {{end}}
            {{end}}
        </div>
        <table>
            <thead>
                <tr>
                    <th>名称</th>
                    <th class="size">大小</th>
                    <th class="time">修改时间</th>
                </tr>
            </thead>
            <tbody>
				{{if not (isRoot .RelPath)}}
                <tr>
                    <td colspan="3"><a href="{{.RelPath}}/../">.. (上级目录)</a></td>
                </tr>
				{{end}}

                {{range .Files}}
                <tr>
                    <td class="{{if .Info.IsDir}}dir-icon{{else}}file-icon{{end}}">
                        <a href="{{.Path}}" {{if .Download}}download{{end}}>{{.Name}}</a>
                    </td>
                    <td class="size">
                        {{if .Info.IsDir}}-{{else}}{{formatBytes .Info.Size}}{{end}}
                    </td>
                    <td class="time">{{formatTime .Info.ModTime}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`))
