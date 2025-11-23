package auxiliary

type Settings struct {
}

func NewSettings() *Settings {
	return &Settings{}
}

func (s *Settings) InitSettings() error {
	return nil
}
