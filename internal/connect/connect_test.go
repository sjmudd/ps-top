package connect

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeDefaults writes an option file and returns its path.
func writeDefaults(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "my.cnf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// emptyHome makes ~/.my.cnf not exist and $USER known.
func emptyHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USER", "login")
}

func TestBuildDSNErrors(t *testing.T) {
	emptyHome(t)
	t.Setenv("MYSQL_DSN", "")
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{"dsn with host and user", Options{DSN: "x", Host: "db1", User: "app"}, "--dsn cannot be used with --host, --user"},
		{"dsn with environment", Options{DSN: "x", UseEnvironment: true}, "--dsn cannot be used with --use-environment"},
		{"dsn with defaults file", Options{DSN: "x", DefaultsFile: "f"}, "--dsn cannot be used with --defaults-file"},
		{"dsn with port, socket, password, askpass", Options{DSN: "x", Port: 1, Socket: "s", Password: "p", Askpass: true},
			"--dsn cannot be used with --port, --socket, --password, --askpass"},
		{"environment with host", Options{UseEnvironment: true, Host: "db1"}, "--use-environment cannot be used with --host"},
		{"environment not set", Options{UseEnvironment: true}, "--use-environment: MYSQL_DSN is not set or empty"},
		{"host and socket", Options{Host: "db1", Socket: "/tmp/mysql.sock"}, "--host and --socket cannot be used together"},
		{"socket and port", Options{Socket: "/tmp/mysql.sock", Port: 3307}, "--socket and --port cannot be used together"},
		{"password and askpass", Options{Password: "pw", Askpass: true}, "--password and --askpass cannot be used together"},
		{"port below range", Options{Port: -1}, "--port -1: not a number from 0 to 65535"},
		{"port above range", Options{Port: 65536}, "--port 65536: not a number from 0 to 65535"},
		{"askpass without reader", Options{Askpass: true}, "--askpass: no way to read the password"},
		{"askpass fails", Options{Askpass: true, ReadPassword: func() (string, error) { return "", errors.New("no terminal") }},
			"--askpass: no terminal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.opts.BuildDSN()
			if err == nil || err.Error() != tt.want {
				t.Errorf("got %q, %v, want error %q", got, err, tt.want)
			}
		})
	}
}

func TestBuildDSNMissingDefaultsFile(t *testing.T) {
	emptyHome(t)
	missing := filepath.Join(t.TempDir(), "missing.cnf")
	if _, err := (Options{DefaultsFile: missing}).BuildDSN(); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got %v, want a file not found error", err)
	}
}

func TestBuildDSN(t *testing.T) {
	emptyHome(t)
	tcpFile := writeDefaults(t, "[client]\nuser=app\npassword=secret\nhost=db1\nport=3307\n")
	socketFile := writeDefaults(t, "[client]\nuser=app\nsocket=/run/mysqld/mysqld.sock\n")
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{"dsn as given", Options{DSN: "u:p@tcp(h:1)/d"}, "u:p@tcp(h:1)/d"},
		{"dsn as given, its own interpolateParams", Options{DSN: "u:p@tcp(h:1)/d?interpolateParams=false"}, "u:p@tcp(h:1)/d?interpolateParams=false"},
		{"no options, no ~/.my.cnf", Options{}, "login@tcp(127.0.0.1:3306)/?interpolateParams=true"},
		{"flags only", Options{Host: "db2", Port: 3308, User: "app", Password: "pw"}, "app:pw@tcp(db2:3308)/?interpolateParams=true"},
		{"socket flag", Options{Socket: "/tmp/mysql.sock", User: "app"}, "app@unix(/tmp/mysql.sock)/?interpolateParams=true"},
		{"file", Options{DefaultsFile: tcpFile}, "app:secret@tcp(db1:3307)/?interpolateParams=true"},
		{"file, host flag keeps file port", Options{DefaultsFile: tcpFile, Host: "db2"}, "app:secret@tcp(db2:3307)/?interpolateParams=true"},
		{"file, port flag", Options{DefaultsFile: tcpFile, Port: 3309}, "app:secret@tcp(db1:3309)/?interpolateParams=true"},
		{"file, user and password flags", Options{DefaultsFile: tcpFile, User: "other", Password: "pw"}, "other:pw@tcp(db1:3307)/?interpolateParams=true"},
		{"file, socket flag replaces host and port", Options{DefaultsFile: tcpFile, Socket: "/tmp/mysql.sock"}, "app:secret@unix(/tmp/mysql.sock)/?interpolateParams=true"},
		{"socket file, host flag replaces socket", Options{DefaultsFile: socketFile, Host: "db2"}, "app@tcp(db2:3306)/?interpolateParams=true"},
		{"socket file, port flag keeps socket", Options{DefaultsFile: socketFile, Port: 3307}, "app@unix(/run/mysqld/mysqld.sock)/?interpolateParams=true"},
		{"askpass replaces the file's password", Options{DefaultsFile: tcpFile, Askpass: true, ReadPassword: func() (string, error) { return "asked", nil }},
			"app:asked@tcp(db1:3307)/?interpolateParams=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.opts.BuildDSN()
			if err != nil || got != tt.want {
				t.Errorf("got %q, %v, want %q", got, err, tt.want)
			}
		})
	}
}

func TestBuildDSNHomeDefaultsFile(t *testing.T) {
	emptyHome(t)
	if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), ".my.cnf"), []byte("[client]\nuser=app\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := (Options{}).BuildDSN()
	if want := "app@tcp(127.0.0.1:3306)/?interpolateParams=true"; err != nil || got != want {
		t.Errorf("got %q, %v, want %q", got, err, want)
	}
}

func TestBuildDSNEnvironment(t *testing.T) {
	t.Setenv("MYSQL_DSN", "u:p@tcp(h:1)/d")
	got, err := (Options{UseEnvironment: true}).BuildDSN()
	if err != nil || got != "u:p@tcp(h:1)/d" {
		t.Errorf("got %q, %v", got, err)
	}
}
