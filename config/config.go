package config

import (
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

func Load(envPrefix, file string, out any) error {
	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	bindEnvs(v, out)

	if file != "" {
		v.SetConfigFile(file)
		_ = v.ReadInConfig()
	}
	return v.Unmarshal(out)
}

func bindEnvs(v *viper.Viper, out any) {
	t := reflect.TypeOf(out)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		_ = v.BindEnv(tag)
	}
}
