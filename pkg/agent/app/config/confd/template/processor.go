package template

func Process(config Config) error {
	tr, err := NewTemplateResource(config)
	if err != nil {
		return err
	}
	return tr.process()
}
