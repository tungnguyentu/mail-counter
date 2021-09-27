package mailcounter

import (
	"log"

	mailcounter "github.com/Tungnt24/mail-counter"
	"github.com/kelseyhightower/envconfig"
)

func Load() mailcounter.Config {
	var cfg mailcounter.Config
	err := envconfig.Process("mail_counter", &cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	return cfg
}
