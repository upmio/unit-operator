package app

func InitAllApp() error {
	for _, api := range grpcApps {
		if err := api.Config(); err != nil {
			return err
		}
	}

	return nil
}
