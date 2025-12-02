package auxiliary

type Settings struct {
}

func NewSettings() (*Settings, error) {
	settings := new(Settings)
	if err := settings.initSettings(); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *Settings) initSettings() error {
	return nil
}
