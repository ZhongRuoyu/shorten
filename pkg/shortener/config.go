package shortener

type Config struct {
	ListenPort int
	Auth       bool
	UrlPrefix  string
	MainPage   string
	CodeLength int
	SqliteDb   string
	LogFile    string
}
