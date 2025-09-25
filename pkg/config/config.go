package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	PGUserName   string `mapstructure:"PG_USERNAME" validate:"required"`
	PGPassword   string `mapstructure:"PG_PASSWORD" validate:"required"`
	PgSSLMode    string `mapstructure:"PG_SSL_MODE" validate:"required"`
	PGDBmsName   string `mapstructure:"PG_DBMS_NAME" validate:"required"`
	PGHost       string `mapstructure:"PG_HOST" validate:"required"`
	PgDriverName string `mapstructure:"PG_DRIVER_NAME" validate:"required"`
	PGDBName     string `mapstructure:"PG_DB_NAME" validate:"required"`
	PgPort       string `mapstructure:"PG_PORT" validate:"required"`

	Host       string `mapstructure:"HOST" validate:"required"`
	ServerPort string `mapstructure:"SERVER_PORT" validate:"required"`
	SMTPemail  string `mapstructure:"EMAIL" validate:"required,email"`
	Password   string `mapstructure:"PASSWORD" validate:"required"`
	AdJWTKey   string `mapstructure:"adminjwtkey" validate:"required"`
	VnJWTKey   string `mapstructure:"vendorjwtkey" validate:"required"`
	BUCKET_ID  string `mapstructure:"BUCKET_ID" validate:""`
	AWS_ACCKEY string `mapstructure:"AWS_ACCKEY" validate:""`
	AWS_REGION string `mapstructure:"AWS_REGION" validate:""`
	AWS_SECKEY string `mapstructure:"AWS_SECKEY" validate:""`
}

var envs = []string{
	"PG_USERNAME", "PG_PASSWORD", "PG_SSL_MODE", "PG_DBMS_NAME", "PG_HOST",
	"PG_DRIVER_NAME", "PG_DB_NAME", "PG_PORT", "HOST", "SERVER_PORT", "EMAIL", "PASSWORD", "adminjwtkey", "VnJWTKey", "RAZORPAY_KEY_ID", "RAZORPAY_SECRET",
	"BUCKET_ID", "AWS_REGION", "AWS_ACCKEY", "AWS_SECKEY",
}

func LoadConfig() (Config, error) {
	var config Config

	viper.SetConfigFile("./pkg/config/.env")
	if err := viper.ReadInConfig(); err != nil {
		return config, err
	}

	viper.AutomaticEnv()

	for _, env := range envs {
		if err := viper.BindEnv(env); err != nil {
			return config, err
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return config, err
	}

	if err := validator.New().Struct(&config); err != nil {
		return config, err
	}

	return config, nil
}
