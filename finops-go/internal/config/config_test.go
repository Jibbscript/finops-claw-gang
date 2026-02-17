package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	clearEnv(t)
	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.Equal(t, ModeStub, cfg.Mode)
	assert.Equal(t, "us-east-1", cfg.AWSRegion)
	assert.Equal(t, "primary", cfg.CURWorkgroup)
}

func TestLoadFromEnv_ProductionValid(t *testing.T) {
	clearEnv(t)
	t.Setenv("FINOPS_MODE", "production")
	t.Setenv("FINOPS_CUR_DATABASE", "cur_db")
	t.Setenv("FINOPS_CUR_TABLE", "cur_table")
	t.Setenv("FINOPS_CUR_OUTPUT_BUCKET", "s3://cur-output")

	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.Equal(t, ModeProduction, cfg.Mode)
	assert.Equal(t, "cur_db", cfg.CURDatabase)
}

func TestLoadFromEnv_ProductionMissingRequired(t *testing.T) {
	clearEnv(t)
	t.Setenv("FINOPS_MODE", "production")

	_, err := LoadFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FINOPS_CUR_DATABASE")
}

func TestLoadFromEnv_InvalidMode(t *testing.T) {
	clearEnv(t)
	t.Setenv("FINOPS_MODE", "invalid")

	_, err := LoadFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid FINOPS_MODE")
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"FINOPS_MODE", "FIXTURES_DIR", "AWS_REGION", "AWS_PROFILE",
		"FINOPS_CROSS_ACCOUNT_ROLE", "FINOPS_CUR_DATABASE", "FINOPS_CUR_TABLE",
		"FINOPS_CUR_WORKGROUP", "FINOPS_CUR_OUTPUT_BUCKET", "FINOPS_KUBECOST_ENDPOINT",
	} {
		// t.Setenv saves the current value and restores it on cleanup.
		// Setting to "" then unsetting ensures the key is absent during the test.
		orig, wasSet := os.LookupEnv(key)
		if wasSet {
			t.Cleanup(func() { os.Setenv(key, orig) })
		} else {
			t.Cleanup(func() { os.Unsetenv(key) })
		}
		os.Unsetenv(key)
	}
}
