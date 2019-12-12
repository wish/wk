package kops

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/wish/wk/pkg/jsonnet"
	"github.com/wish/wk/pkg/opa"
	"github.com/wish/wk/pkg/util"
	_ "k8s.io/kops/util/pkg/vfs"
)

type channelItem struct {
	path string
	hash string
}

type channelItems []channelItem

func (a channelItems) Len() int           { return len(a) }
func (a channelItems) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a channelItems) Less(i, j int) bool { return a[i].path < a[j].path }

type errors struct {
	inner []error
	mu    sync.Mutex
}

func (e *errors) Add(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.inner = append(e.inner, err)
}

func (e *errors) Get() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.inner
}

const channelPrefix = `kind: Addons
metadata:
  creationTimestamp: null
  name: wish
spec:
  addons:
`

const addonStr = `     - manifest: %v
       name: %v
       version: 0.1.0
       id: %v
`

func ChannelsApply(ctx context.Context, file, dryFile string, opaQuery *opa.OPA) error {
	conf, err := util.GetConfig(file)
	if err != nil {
		return err
	}
	ctxDir := conf.ContextDir

	cluster, _, err := jsonnet.ExpandCluster(ctx, file)
	if err != nil {
		return err
	}
	if cluster.Kops == nil {
		return fmt.Errorf("kops configuration is missing")
	}

	chItemsMu := sync.Mutex{}
	chItems := channelItems{}
	mkdirMu := sync.Mutex{}

	errors := errors{
		inner: make([]error, 0),
		mu:    sync.Mutex{},
	}

	for _, channel := range cluster.Kops.Channels {
		var regex *regexp.Regexp
		if channel.FileWhitelistRegexp != nil {
			regex, err = regexp.Compile(*channel.FileWhitelistRegexp)
			if err != nil {
				return err
			}
		} else {
			regex = regexp.MustCompile("\\.jsonnet$")
		}

		if channel.Folder != "" {
			fold := filepath.Join(ctxDir, channel.Folder)
			files := []string{}
			if err = filepath.Walk(fold, func(path string, info os.FileInfo, err error) error {
				if regex.Match([]byte(path)) {
					files = append(files, path)
				}
				return nil
			}); err != nil {
				return err
			}

			wg := &sync.WaitGroup{}
			for _, path := range files {
				wg.Add(1)
				go func(path string, wg *sync.WaitGroup) {
					defer wg.Done()
					empty, outFile, hsh, err2 := jsonnet.ExpandAppFile(ctx, path, file)
					if err2 != nil {
						errors.Add(err2)
						return
					}

					if !empty && opaQuery != nil {
						accepted, issues, err2 := opaQuery.RunFile(outFile)
						if err2 != nil {
							errors.Add(err2)
							return
						}
						if !accepted {
							for _, issue := range issues {
								errors.Add(fmt.Errorf("Issue with file %v: %v", path, issue))
							}
							return
						}
					}

					if !empty {
						if dryFile != "" {
							tfile := filepath.Join(dryFile, path[len(fold):])
							tfile = strings.ReplaceAll(tfile, ".jsonnet", ".json")
							mkdirMu.Lock()
							if err2 := os.MkdirAll(filepath.Dir(tfile), os.ModePerm); err2 != nil {
								errors.Add(err2)
								mkdirMu.Unlock()
								return
							}
							mkdirMu.Unlock()
							if err2 := CopyFile(outFile, tfile); err2 != nil {
								errors.Add(err2)
								return
							}
							chItemsMu.Lock()
							chItems = append(chItems, channelItem{
								strings.ReplaceAll(path[len(fold)+1:], ".jsonnet", ".json"), hsh,
							})
							chItemsMu.Unlock()
						}
					}
				}(path, wg)
			}
			wg.Wait()
			errs := errors.Get()
			if len(errs) > 0 {
				for _, err := range errs {
					fmt.Printf("%v\n", err)

				}
				return fmt.Errorf("%v errors encountered compiling channel %v", len(errs), channel.Name)
			}
		} else {
			// TODO(tvi): Add more supported types.
		}
	}

	if dryFile != "" {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dryFile, "channel.yaml")), os.ModePerm); err != nil {
			return err
		}
		sort.Sort(chItems)

		out := channelPrefix
		for _, it := range chItems {
			out += fmt.Sprintf(addonStr, it.path, pathToName(it.path), it.hash)
		}

		if err := ioutil.WriteFile(filepath.Join(dryFile, "channel.yaml"), []byte(out), 0644); err != nil {
			log.Fatal(err)
		}
		return nil
	}
	return nil
}

func pathToName(path string) string {
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ReplaceAll(path, ".", "-")
	return strings.ReplaceAll(path, "_", "")
}
