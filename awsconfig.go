package awsclient

type AwsConfig struct {
	Profile         string
	AccessKeyId     string
	AccessKeySecret string
	Region          string
}

type Config []AwsConfig

func (c *Config) Init() error {
	for _, awscfg := range []AwsConfig(*c) {
		SetSession(awscfg.Profile, awscfg)
	}

	return nil
}
