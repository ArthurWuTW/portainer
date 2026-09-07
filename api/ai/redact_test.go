package ai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSensitiveKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"DATABASE_PASSWORD", true},
		{"API_KEY", true},
		{"OPENAI_APIKEY", true},
		{"AUTH_TOKEN", true},
		{"PRIVATE_KEY", true},
		{"DB_CREDENTIAL", true},
		{"SECRET", true},
		{"PATH", false},
		{"PORT", false},
		{"NGINX_HOST", false},
	}

	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			require.Equal(t, c.want, isSensitiveKey(c.key))
		})
	}
}

func TestRedactEnv(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Nil(t, redactEnv(nil))
	})

	t.Run("redacts sensitive keys", func(t *testing.T) {
		in := []string{
			"DATABASE_PASSWORD=supersecret",
			"API_KEY=abc123",
			"PATH=/usr/bin",
			"PORT=8080",
		}
		out := redactEnv(in)
		require.Equal(t, []string{
			"DATABASE_PASSWORD=" + redactedValue,
			"API_KEY=" + redactedValue,
			"PATH=/usr/bin",
			"PORT=8080",
		}, out)
	})

	t.Run("entry without equals is fully redacted", func(t *testing.T) {
		out := redactEnv([]string{"JUSTAVALUE"})
		require.Equal(t, []string{redactedValue}, out)
	})
}

func TestRedactLabels(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Nil(t, redactLabels(nil))
	})

	t.Run("redacts sensitive values", func(t *testing.T) {
		in := map[string]string{
			"com.docker.traffic": "80",
			"db_password":        "hunter2",
		}
		out := redactLabels(in)
		require.Equal(t, redactedValue, out["db_password"])
		require.Equal(t, "80", out["com.docker.traffic"])
	})
}

func TestSanitizeURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"tcp://192.168.1.100:2376", "tcp://192.168.1.100:2376"},
		{"tcp://user:pass@192.168.1.100:2376", "tcp://192.168.1.100:2376"},
		{"unix:///var/run/docker.sock", "unix:///var/run/docker.sock"},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			require.Equal(t, c.want, sanitizeURL(c.in))
		})
	}
}
