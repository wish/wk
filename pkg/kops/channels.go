package kops

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wish/wk/pkg/jsonnet"
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

func ChannelsApply(ctx context.Context, file, dryFile string) error {
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

	chItems := channelItems{}

	for _, channel := range cluster.Kops.Channels {
		if channel.Folder != "" {
			fold := filepath.Join(ctxDir, channel.Folder)
			files := []string{}
			if err = filepath.Walk(fold, func(path string, info os.FileInfo, err error) error {
				if strings.HasSuffix(path, ".jsonnet") {
					files = append(files, path)
				}
				return nil
			}); err != nil {
				return err
			}

			var err error
			wg := &sync.WaitGroup{}
			for _, path := range files {
				wg.Add(1)
				go func(path string, wg *sync.WaitGroup) {
					empty, outFile, hsh, err2 := jsonnet.ExpandAppFile(ctx, path, file)
					if err2 != nil {
						err = err2
					}

					if !empty {
						if dryFile != "" {
							tfile := filepath.Join(dryFile, path[len(fold):])
							tfile = strings.ReplaceAll(tfile, ".jsonnet", ".json")
							if err2 := os.MkdirAll(filepath.Dir(tfile), os.ModePerm); err2 != nil {
								err = err2
							}
							if err2 := CopyFile(outFile, tfile); err2 != nil {
								err = err2
							}
							chItems = append(chItems, channelItem{
								strings.ReplaceAll(path[len(fold)+1:], ".jsonnet", ".json"), hsh,
							})
						}
					}
					wg.Done()
				}(path, wg)
			}
			wg.Wait()
			if err != nil {
				return err
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
