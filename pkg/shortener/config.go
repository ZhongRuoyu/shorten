package shortener

type Config struct {
	ListenPort int
	UrlPrefix  string
	MainPage   string
	CodeLength int
	SqliteDb   string
	LogFile    string
}
