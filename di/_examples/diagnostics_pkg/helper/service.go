package helper

type Config struct {
	Name string
}

type Service struct {
	Config Config
}

func NewService(cfg Config) *Service {
	return &Service{Config: cfg}
}
