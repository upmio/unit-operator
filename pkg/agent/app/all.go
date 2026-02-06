package app

import (
	"context"
	"fmt"
	"sync"
)

func InitAllApp(ctx context.Context, wg *sync.WaitGroup) error {
	for _, api := range grpcApps {
		if err := api.Config(); err != nil {
			return err
		}
	}

	for name, app := range daemonApps {
		err := app.Config()
		if err != nil {
			return fmt.Errorf("config daemon app %s error %s", name, err)
		}

		app.Registry(ctx, wg)
	}

	return nil
}
