package config

import (
	"hawkeye/database"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const (
	ProviderAWS = "aws"
)

const (
	PrefixSSM = "ssm://"
)

type AppConfig struct {
	Region                   string `mapstructure:"region"`
	Provider                 string `mapstructure:"provider"`
	AwsProfile               string `mapstructure:"aws_profile"`
	AwsSSMParamPrefix        string `mapstructure:"aws_ssm_param_prefix"`
	RedisHost                string `mapstructure:"redis_url"`
	NotificationServiceURL   string `mapstructure:"notification_api"`
	NotificationServiceToken string `mapstructure:"notification_secret"`
	MonitorConfigFile        string `mapstructure:"monitor_config_file"`
	Environment              string `mapstricture:"environment"`
	ServiceName              string `mapstructure:"service_name"`
}

var (
	once sync.Once
	_cfg AppConfig
)

func ReadConfig() AppConfig {
	once.Do(func() {
		_cfg = read()
	})

	return _cfg
}

func read() AppConfig {
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}

	defaultEnvFile := ".env"
	envFile := "." + strings.ToLower(env) + ".env"

	if _, err := os.Stat(defaultEnvFile); err != nil {
		log.Fatal(".env file must be present")
	}

	if _, err := os.Stat(envFile); err != nil {
		log.Println("failed to find ", envFile, " resorting to ", defaultEnvFile)
		envFile = defaultEnvFile
	}

	log.Println("reading env from", envFile)

	viper.AddConfigPath(".")
	viper.SetConfigFile(envFile)

	viper.AutomaticEnv()

	cfg := AppConfig{}

	if err := viper.ReadInConfig(); err != nil {
		return cfg
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal("failed to read env config ", err)
	}

	if cfg.Environment == "" {
		cfg.Environment = "dev"
	}
	cfg.Environment = strings.ToLower(cfg.Environment)

	if cfg.Provider == ProviderAWS {
		cfg = ResolveSSMParams(cfg)
	}

	return cfg

}

func Get() AppConfig {
	if _cfg.Provider == "" {
		log.Fatal("config load failed")
	}

	return _cfg
}

func (c AppConfig) ValidateConnections() {
	_ = database.NewRedisClient(c.RedisHost)
}

func ResolveSSMParams(cfg AppConfig) AppConfig {
	var c = cfg

	if cfg.AwsProfile == "" {
		c.AwsProfile = "default"
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            aws.Config{Region: aws.String(cfg.Region)},
		SharedConfigState: session.SharedConfigEnable,
		Profile:           cfg.AwsProfile,
	})

	if err != nil {
		log.Fatal("failed to make session from aws ", err)
	}

	sm := ssm.New(sess, aws.NewConfig().WithRegion(cfg.Region).WithMaxRetries(10))

	fromSSM := WithSSM(sm, cfg.AwsSSMParamPrefix, cfg.Environment)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		c.RedisHost = fromSSM(cfg.RedisHost)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		c.NotificationServiceToken = fromSSM(cfg.NotificationServiceToken)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		c.NotificationServiceURL = fromSSM(cfg.NotificationServiceURL)
	}()

	wg.Wait()

	return c
}

type Valuer func(value string) string

func WithSSM(sm *ssm.SSM, prefix, env string) Valuer {
	partPrefix := filepath.Join("/", prefix, env)

	return func(value string) string {
		if !strings.HasPrefix(value, PrefixSSM) {
			return value
		}

		value = strings.TrimPrefix(value, PrefixSSM)
		ssmKey := filepath.Join(partPrefix, value)

		out, err := sm.GetParameter(&ssm.GetParameterInput{
			Name:           aws.String(ssmKey),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			log.Fatal("failed to get ", ssmKey, " from ssm ", err)
		}

		if out.Parameter == nil || (out.Parameter != nil && out.Parameter.Value == nil) {
			log.Fatal("parameter not found in ssm ", ssmKey)
		}

		return *out.Parameter.Value
	}
}
