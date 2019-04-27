package kops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func Update(ctx context.Context, file string) (err error) {
	out, err := Edit(ctx, file, "", "")
	if err != nil {
		return fmt.Errorf("could not render jsonnet: %v", err)
	}
	if out.Kops == nil {
		return fmt.Errorf("kops configuration is missing")
	}

	kopsEnv := os.Environ()
	for k, v := range out.Kops.Env {
		kopsEnv = append(kopsEnv, fmt.Sprintf("%v=%v", k, v))
	}

	updateCmd := exec.CommandContext(ctx, "kops",
		"update", "cluster", "--name="+out.Name, "-v10", "--yes")
	updateCmd.Stdout, updateCmd.Stderr = os.Stdout, os.Stderr
	updateCmd.Env = kopsEnv
	err = updateCmd.Run()
	if err != nil {
		return fmt.Errorf("could not edit cluster: %v", err)
	}
	return nil
}

func Delete(ctx context.Context, file string) (err error) {
	out, err := Edit(ctx, file, "", "")
	if err != nil {
		return fmt.Errorf("could not render jsonnet: %v", err)
	}
	if out.Kops == nil {
		return fmt.Errorf("kops configuration is missing")
	}
	for _, c := range out.Kops.Channels {
		err := deleteChannel(ctx, c)
		if err != nil {
			return fmt.Errorf("could not delete channel: %v", err)
		}
	}

	kopsEnv := os.Environ()
	for k, v := range out.Kops.Env {
		kopsEnv = append(kopsEnv, fmt.Sprintf("%v=%v", k, v))
	}

	updateCmd := exec.CommandContext(ctx, "kops",
		"delete", "cluster", "--name="+out.Name, "-v10", "--yes")
	updateCmd.Stdout, updateCmd.Stderr = os.Stdout, os.Stderr
	updateCmd.Env = kopsEnv
	err = updateCmd.Run()
	if err != nil {
		return fmt.Errorf("could not edit cluster: %v", err)
	}
	return nil
}
