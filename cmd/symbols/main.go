// Command symbols is a service that serves code symbols (functions, variables, etc.) from a repository at a
// specific commit.
package main

//docker:run apk --no-cache add curl jansson-dev libseccomp-dev linux-headers autoconf pkgconfig make automake gcc g++ binutils && curl https://codeload.github.com/universal-ctags/ctags/tar.gz/7918d19fe358fae9bad1c264c4f5dc2dcde5cece | tar xz -C /tmp && cd /tmp/ctags-7918d19fe358fae9bad1c264c4f5dc2dcde5cece && ./autogen.sh && LDFLAGS=-static ./configure --program-prefix=universal- --enable-json --enable-seccomp && make -j8 && make install && cd && rm -rf /tmp/ctags-7918d19fe358fae9bad1c264c4f5dc2dcde5cece && apk --no-cache --purge del curl jansson-dev libseccomp-dev linux-headers autoconf pkgconfig make automake gcc g++ binutils

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	log15 "gopkg.in/inconshreveable/log15.v2"

	"github.com/sourcegraph/sourcegraph/cmd/symbols/internal/pkg/ctags"
	"github.com/sourcegraph/sourcegraph/cmd/symbols/internal/symbols"
	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/debugserver"
	"github.com/sourcegraph/sourcegraph/pkg/env"
	"github.com/sourcegraph/sourcegraph/pkg/gitserver"
	"github.com/sourcegraph/sourcegraph/pkg/tracer"
)

var (
	cacheDir       = env.Get("CACHE_DIR", "/tmp/symbols-cache", "directory to store cached symbols")
	cacheSizeMB    = env.Get("SYMBOLS_CACHE_SIZE_MB", "0", "maximum size of the disk cache in megabytes")
	ctagsProcesses = env.Get("CTAGS_PROCESSES", strconv.Itoa(runtime.NumCPU()), "number of ctags child processes to run")
	ctagsCommand   = env.Get("CTAGS_COMMAND", "universal-ctags", "ctags command (should point to universal-ctags executable compiled with JSON and seccomp support)")
)

func main() {
	env.Lock()
	env.HandleHelpFlag()
	log.SetFlags(0)
	tracer.Init("symbols")

	// Filter log output by level.
	lvl, err := log15.LvlFromString(env.LogLevel)
	if err == nil {
		log15.Root().SetHandler(log15.LvlFilterHandler(lvl, log15.StderrHandler))
	}

	go debugserver.Start()

	service := symbols.Service{
		FetchTar: func(ctx context.Context, repo gitserver.Repo, commit api.CommitID) (io.ReadCloser, error) {
			return gitserver.FetchTar(ctx, gitserver.DefaultClient, repo, commit)
		},
		NewParser: func() (ctags.Parser, error) {
			parser, err := ctags.NewParser(ctagsCommand)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("command: %s", ctagsCommand))
			}
			return parser, nil
		},
		Path: cacheDir,
	}
	if mb, err := strconv.ParseInt(cacheSizeMB, 10, 64); err != nil {
		log.Fatalf("Invalid SYMBOLS_CACHE_SIZE_MB: %s", err)
	} else {
		service.MaxCacheSizeBytes = mb * 1000 * 1000
	}
	service.NumParserProcesses, err = strconv.Atoi(ctagsProcesses)
	if err != nil {
		log.Fatalf("Invalid CTAGS_PROCESSES: %s", err)
	}
	if err := service.Start(); err != nil {
		log.Fatalln("Start:", err)
	}
	handler := nethttp.Middleware(opentracing.GlobalTracer(), service.Handler())

	addr := ":3184"
	server := &http.Server{Addr: addr, Handler: handler}
	go shutdownOnSIGINT(server)

	log15.Info("symbols: listening", "addr", ":3184")
	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func shutdownOnSIGINT(s *http.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := s.Shutdown(ctx)
	if err != nil {
		log.Fatal("graceful server shutdown failed, will exit:", err)
	}
}

type badRequestError struct{ msg string }

func (e badRequestError) Error() string    { return e.msg }
func (e badRequestError) BadRequest() bool { return true }
