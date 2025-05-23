package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Host       string
	Port       string
	SAMLConfig SAMLConfig
	LDAPConfig LDAPConfig
	DBConfig   DBConfig
	AuthConfig AuthConfig
}

type SAMLConfig struct {
	EntityID    string
	IDPMetadata string
	KeyFile     string
	CertFile    string
	ACSUrl      string
}

type LDAPConfig struct {
	Host     string
	Port     int
	BaseDN   string
	BindDN   string
	BindPass string
	UseSSL   bool
}

type DBConfig struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
}

type AuthConfig struct {
	JWTSecret      string
	JWTExpiryHours int
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	return &Config{
		Host: viper.GetString("HOST"),
		Port: viper.GetString("PORT"),
		SAMLConfig: SAMLConfig{
			EntityID:    viper.GetString("SAML_ENTITY_ID"),
			IDPMetadata: viper.GetString("SAML_IDP_METADATA"),
			KeyFile:     viper.GetString("SAML_SP_KEY"),
			CertFile:    viper.GetString("SAML_SP_CERT"),
			ACSUrl:      viper.GetString("SAML_ACS_URL"),
		},
		LDAPConfig: LDAPConfig{
			Host:     viper.GetString("LDAP_HOST"),
			Port:     viper.GetInt("LDAP_PORT"),
			BaseDN:   viper.GetString("LDAP_BASEDN"),
			BindDN:   viper.GetString("LDAP_BIND_DN"),
			BindPass: viper.GetString("LDAP_BIND_PASS"),
			UseSSL:   viper.GetBool("LDAP_USE_SSL"),
		},
		DBConfig: DBConfig{
			DBHost:     viper.GetString("DB_HOST"),
			DBPort:     viper.GetInt("DB_PORT"),
			DBUser:     viper.GetString("DB_USER"),
			DBPassword: viper.GetString("DB_PASSWORD"),
			DBName:     viper.GetString("DB_NAME"),
		},
		AuthConfig: AuthConfig{
			JWTSecret:      viper.GetString("JWT_SECRET"),
			JWTExpiryHours: viper.GetInt("JWT_EXPIRY_HOURS"),
		},
	}, nil
}

func (c *Config) GetDBUrl() string {
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		c.DBConfig.DBUser,
		c.DBConfig.DBPassword,
		c.DBConfig.DBHost,
		c.DBConfig.DBPort,
		c.DBConfig.DBName,
	)

	if !strings.Contains(url, "sslmode=") {
		if strings.Contains(url, "localhost") || strings.Contains(url, "127.0.0.1") {
			url += "?sslmode=disable"
		} else {
			url += "?sslmode=prefer"
		}
	}

	return url
}

func (c *Config) GetBaseURL() string {
	return fmt.Sprintf("http://%s:%s", c.Host, c.Port)
	// return fmt.Sprintf("http://localhost:8080")
}
