package discovery

// Keys holds API keys for various services
type Keys struct {
	CensysToken     string
	CensysSecret    string
	Shodan          string
	FofaEmail       string
	FofaKey         string
	QuakeToken      string
	HunterToken     string
	ZoomEyeHost     string
	ZoomEyeToken    string
	NetlasToken     string
	CriminalIPToken string
	PublicwwwToken  string
	HunterHowToken  string
	GoogleKey       string
	GoogleCX        string
}

// Empty checks if the keys struct has any keys set
func (keys Keys) Empty() bool {
	return keys.CensysToken == "" &&
		keys.CensysSecret == "" &&
		keys.Shodan == "" &&
		keys.FofaEmail == "" &&
		keys.FofaKey == "" &&
		keys.QuakeToken == "" &&
		keys.HunterToken == "" &&
		keys.ZoomEyeHost == "" &&
		keys.ZoomEyeToken == "" &&
		keys.NetlasToken == "" &&
		keys.CriminalIPToken == "" &&
		keys.PublicwwwToken == "" &&
		keys.HunterHowToken == "" &&
		keys.GoogleKey == "" &&
		keys.GoogleCX == ""
}
