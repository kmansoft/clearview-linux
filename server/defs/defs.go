package defs

type AccountId string

type Flags struct {
	ListenAddr string
	ListenPort int

	Verbose       bool
	SleepInterval int

	HttpWebCvDir string

	InfluxDbUri  string
	InfluxDbName string

	MongoServerName string
	MongoDbName     string

	ConfigFileName string

	DemoMode bool
}

const (
	TECH_INFLUX = "iflx"
)
