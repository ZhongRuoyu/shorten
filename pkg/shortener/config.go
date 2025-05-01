package shortener

type Config struct {
	Auth       bool
	ListenPort int
	UrlPrefix  string
	MainPage   string
	CodeLength int
	SqliteDb   string
	LogFile    string
}
