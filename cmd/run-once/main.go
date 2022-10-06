package main

import (
	"encoding/json"
	"flag"
	"grail/sysinfra/cfg/config"
	"grail/sysinfra/cfg/log"
)

func main() {

	buildNumber := flag.String("build_number", "", "host name")
	flag.Parse() // add this line

	conf, err := config.Init(config.Defaults(
		config.Set(config.COMMIT, "xe32sdf"),
	))
	if *buildNumber != "" {
		conf.Build.BuildNumber = *buildNumber
	}

	if err != nil {
		log.Fatalf("error initializing configuration: %v", err)
	}
	res2B, err := json.Marshal(conf)
	if err != nil {
		log.Fatalf("error initializing configuration: %v", err)
	}
	log.Printf("Config is \n\t%s\n", res2B)
}
