package config

import (
	"flag"
	"os"
	"strconv"
)

type Parameters struct {
	AddressHTTP string
	AddrDB      string
	SystemAddr  string
	Interval    int
}

func NewParameters() *Parameters {
	return &Parameters{}
}
func (p *Parameters) getParameters() {
	addr := flag.String("a", "localhost:8080", "address HTTP")
	addrDB := flag.String("d", "", "String with database connection address")
	systemAddr := flag.String("r", "", "system address")
	interval := flag.Int("p", 2, "how often will I contact the database?")

	flag.Parse()
	p.AddressHTTP = *addr
	p.AddrDB = *addrDB
	p.SystemAddr = *systemAddr
	p.Interval = *interval

}
func (p *Parameters) getParametersEnvironmentVariables() {
	addr := os.Getenv("RUN_ADDRESS")
	if addr != "" {
		p.AddressHTTP = addr
	}
	addrDB := os.Getenv("DATABASE_URI")
	if addrDB != "" {
		p.AddrDB = addrDB
	}
	systemAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")

	if systemAddr != "" {
		p.SystemAddr = systemAddr
	}
	intervalStr := os.Getenv("INTERVAL")
	interval, _ := strconv.Atoi(intervalStr)
	if intervalStr != "" {
		p.Interval = interval
	}
}

func (p *Parameters) Get() {
	p.getParameters()
	p.getParametersEnvironmentVariables()
}
