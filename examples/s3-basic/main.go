package main

import (
	"fmt"
	"os"

	us "github.com/bronystylecrazy/ultrastructure"
	uscmd "github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/bronystylecrazy/ultrastructure/web"
)

func main() {
	if err := us.New(
		uscmd.Run(
			web.Init(),
			di.Provide(NewUploadHandler),
			di.Invoke(RequireBucket),
		),
	).Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func RequireBucket(cfg s3.Config) error {
	if cfg.Bucket == "" {
		return fmt.Errorf("storage.s3.bucket is required")
	}
	return nil
}
