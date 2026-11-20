package etc

import (
	"testing"

	"github.com/xbaseio/xbase/core/value"
)

func TestOverrideWithEnvironment(t *testing.T) {
	t.Setenv("XBASE_CONFIG_LOCATE_REDIS_PASSWORD", "from-env")
	got := overrideWithEnvironment("etc.locate.redis.password", value.NewValue("from-file"))
	if got.String() != "from-env" {
		t.Fatalf("password = %q, want environment value", got.String())
	}
}

func TestOverrideWithEnvironmentArray(t *testing.T) {
	t.Setenv("XBASE_CONFIG_LOG_OUTPUT_PATHS", `["stdout","/tmp/xgame.log"]`)
	got := overrideWithEnvironment("etc.log.outputPaths", value.NewValue([]string{"stderr"}))
	paths := got.Strings()
	if len(paths) != 2 || paths[0] != "stdout" || paths[1] != "/tmp/xgame.log" {
		t.Fatalf("output paths = %#v", paths)
	}
}

func TestConfigEnvName(t *testing.T) {
	if got := configEnvName("etc.eventbus.nats.timeout"); got != "XBASE_CONFIG_EVENTBUS_NATS_TIMEOUT" {
		t.Fatalf("environment name = %q", got)
	}
}
