package filen

import (
	"github.com/rclone/rclone/fstest/fstests"
	"testing"
)

func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestFilen:",
		NilObject:  (*File)(nil),
	})
}
