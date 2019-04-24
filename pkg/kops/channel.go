package kops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wish/wk/pkg/util"

	"k8s.io/kops/util/pkg/vfs"
)

func generateChannel(ctx context.Context, c Channel, file string) error {
	bb := bytes.NewBufferString("std.flattenArrays([\n")
	for _, a := range c.Apps {
		switch a.Type {
		case "file":
			c, err := util.GetConfig()
			if err != nil {
				return err
			}
			fmt.Fprintf(bb, "	import '%v',\n", filepath.Join(c.ContextDir, a.File.Path))
		default:
		}
	}
	fmt.Fprintf(bb, "])\n")

	cmd := exec.CommandContext(ctx, "jsonnet", "-y", "--ext-code", "cluster={data:1}", "-")
	hashB := bytes.NewBufferString("")
	cmd.Stdin = bb
	cmd.Stdout = hashB
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(file)
	if err := cmd.Run(); err != nil {
		return err
	}

	yaml := bytes.NewBuffer([]byte{})
	fmt.Fprintf(yaml, channelTemplate,
		c.Name, c.Name, c.Name,
		sha256.Sum256(hashB.Bytes()))

	dest, err := vfs.Context.BuildVfsPath(c.Path)
	if err != nil {
		return err
	}

	if err := dest.Join(c.Name+".yaml").WriteFile(bytes.NewReader(yaml.Bytes()), nil); err != nil {
		return err
	}
	if err := dest.Join(c.Name+".json").WriteFile(bytes.NewReader(hashB.Bytes()), nil); err != nil {
		return err
	}

	return nil
}
