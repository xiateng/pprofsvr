package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/pprof/driver"
)

type nullUI struct{}

func (n nullUI) ReadLine(prompt string) (string, error) {
	return "", io.EOF
}

func (n nullUI) Print(i ...interface{}) {
	fmt.Fprint(os.Stdout, i...)
}

func (n nullUI) PrintErr(i ...interface{}) {
	fmt.Fprint(os.Stderr, i...)
}

func (n nullUI) IsTerminal() bool {
	return true
}

func (n nullUI) WantBrowser() bool {
	return true
}

func (n nullUI) SetAutoComplete(func(string) string) {
}

var _ driver.UI = (*nullUI)(nil)

type PProfHandler struct {
	args *driver.HTTPServerArgs
	time time.Time
}

var (
	profileMap     sync.Map
	mu             sync.Mutex
	profileTTL     time.Duration = 30 * time.Minute // 设置profile缓存30分钟
	watcherRunning bool
)

// 新增清理函数
func cleanupProfiles() {
	for {
		time.Sleep(5 * time.Minute) // 每5分钟检查一次
		profileMap.Range(func(key, value interface{}) bool {
			if ph, ok := value.(*PProfHandler); ok {
				if time.Since(ph.time) > profileTTL {
					profileMap.Delete(key)
				}
			}
			return true
		})
	}
}

// 修改后的getHandler函数
func getHandler(fp string) (*PProfHandler, error) {
	// 检查文件修改时间
	info, err := os.Stat(fp)
	if err != nil {
		return nil, err
	}

	if v, loaded := profileMap.Load(fp); loaded {
		ph := v.(*PProfHandler)
		// 检查文件是否被修改
		if info.ModTime().After(ph.time) {
			profileMap.Delete(fp) // 文件已修改，删除旧缓存
		} else {
			return ph, nil
		}
	}

	mu.Lock()
	defer mu.Unlock()

	// 再次检查，防止并发创建
	if v, loaded := profileMap.Load(fp); loaded {
		return v.(*PProfHandler), nil
	}

	ph := &PProfHandler{}
	opts := &driver.Options{
		UI: &nullUI{},
		HTTPServer: func(args *driver.HTTPServerArgs) error {
			ph.time = time.Now() // 记录加载时间
			ph.args = args
			return nil
		},
		HTTPTransport: http.DefaultTransport,
		Flagset:       NewGoFlags([]string{"-http", ":8888", "--no_browser", fp}),
	}

	if err := driver.PProf(opts); err != nil {
		return nil, err
	}

	profileMap.Store(fp, ph)
	return ph, nil
}

var (
	repoPath string
	addr     string
)

func init() {
	flag.StringVar(&repoPath, "p", "", "repository path")
	flag.StringVar(&addr, "addr", "", "listen addr, default: :26817")
}

func main() {
	flag.Parse()
	if repoPath == "" {
		repoPath = "."
	}
	if addr == "" {
		addr = ":26817"
	}

	// 在main函数开始时启动清理goroutine
	go cleanupProfiles()

	r := gin.Default()

	fs := gin.Dir(repoPath, true)

	fileServer := http.FileServer(fs)

	// Register GET and HEAD handlers
	r.GET("/*filepath", func(c *gin.Context) {
		if before, after, ok := strings.Cut(c.Request.URL.Path, "/ui"); ok {
			// load http handlers
			fp := filepath.Join(repoPath, before)
			ph, err := getHandler(fp)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			if after == "" {
				after = "/"
			}

			if handler, ok := ph.args.Handlers[after]; ok {
				handler.ServeHTTP(c.Writer, c.Request)
				return
			}
		} else {
			file := c.Param("filepath")
			// Check if file exists and/or if we have permission to access it
			f, err := fs.Open(file)
			if err != nil {
				c.Writer.WriteHeader(http.StatusNotFound)
				return
			}
			if info, err := f.Stat(); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			} else {
				if !info.IsDir() {
					fp := filepath.Join(repoPath, before)
					_, err = getHandler(fp)
					if err != nil {
						c.AbortWithError(http.StatusInternalServerError, err)
						return
					}
					c.Redirect(http.StatusFound, c.Request.URL.Path+"/ui/")
				}
			}

			f.Close()

			fileServer.ServeHTTP(c.Writer, c.Request)
		}
	})

	r.Run(addr)
}
