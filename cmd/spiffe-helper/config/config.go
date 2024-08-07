package config

import (
	"errors"
	"os"

	"github.com/hashicorp/hcl"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spiffe-helper/pkg/sidecar"
)

const (
	defaultAgentAddress = "/tmp/spire-agent/public/api.sock"
)

type Config struct {
	AddIntermediatesToBundle           bool   `hcl:"add_intermediates_to_bundle"`
	AddIntermediatesToBundleDeprecated bool   `hcl:"addIntermediatesToBundle"`
	AgentAddress                       string `hcl:"agent_address"`
	AgentAddressDeprecated             string `hcl:"agentAddress"`
	Cmd                                string `hcl:"cmd"`
	CmdArgs                            string `hcl:"cmd_args"`
	CmdArgsDeprecated                  string `hcl:"cmdArgs"`
	CertDir                            string `hcl:"cert_dir"`
	CertDirDeprecated                  string `hcl:"certDir"`
	ExitWhenReady                      bool   `hcl:"exit_when_ready"`
	IncludeFederatedDomains            bool   `hcl:"include_federated_domains"`
	RenewSignal                        string `hcl:"renew_signal"`
	RenewSignalDeprecated              string `hcl:"renewSignal"`

	// x509 configuration
	SVIDFileName                 string `hcl:"svid_file_name"`
	SVIDFileNameDeprecated       string `hcl:"svidFileName"`
	SVIDKeyFileName              string `hcl:"svid_key_file_name"`
	SVIDKeyFileNameDeprecated    string `hcl:"svidKeyFileName"`
	SVIDBundleFileName           string `hcl:"svid_bundle_file_name"`
	SVIDBundleFileNameDeprecated string `hcl:"svidBundleFileName"`

	// JWT configuration
	JWTSVIDs          []JWTConfig `hcl:"jwt_svids"`
	JWTBundleFilename string      `hcl:"jwt_bundle_file_name"`
}

type JWTConfig struct {
	JWTAudience     string `hcl:"jwt_audience"`
	JWTSVIDFilename string `hcl:"jwt_svid_file_name"`
}

// ParseConfig parses the given HCL file into a Config struct
func ParseConfig(file string) (*Config, error) {
	sidecarConfig := new(Config)

	// Read HCL file
	dat, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Parse HCL
	if err := hcl.Decode(sidecarConfig, string(dat)); err != nil {
		return nil, err
	}

	return sidecarConfig, nil
}

func ValidateConfig(c *Config, exitWhenReady bool, log logrus.FieldLogger) error {
	if err := validateOSConfig(c); err != nil {
		return err
	}
	if c.AgentAddressDeprecated != "" {
		if c.AgentAddress != "" {
			return errors.New("use of agent_address and agentAddress found, use only agent_address")
		}
		log.Warn(getWarning("agentAddress", "agent_address"))
		c.AgentAddress = c.AgentAddressDeprecated
	}

	if c.CmdArgsDeprecated != "" {
		if c.CmdArgs != "" {
			return errors.New("use of cmd_args and cmdArgs found, use only cmd_args")
		}
		log.Warn(getWarning("cmdArgs", "cmd_args"))
		c.CmdArgs = c.CmdArgsDeprecated
	}

	if c.CertDirDeprecated != "" {
		if c.CertDir != "" {
			return errors.New("use of cert_dir and certDir found, use only cert_dir")
		}
		log.Warn(getWarning("certDir", "cert_dir"))
		c.CertDir = c.CertDirDeprecated
	}

	if c.SVIDFileNameDeprecated != "" {
		if c.SVIDFileName != "" {
			return errors.New("use of svid_file_name and svidFileName found, use only svid_file_name")
		}
		log.Warn(getWarning("svidFileName", "svid_file_name"))
		c.SVIDFileName = c.SVIDFileNameDeprecated
	}

	if c.SVIDKeyFileNameDeprecated != "" {
		if c.SVIDKeyFileName != "" {
			return errors.New("use of svid_key_file_name and svidKeyFileName found, use only svid_key_file_name")
		}
		log.Warn(getWarning("svidKeyFileName", "svid_key_file_name"))
		c.SVIDKeyFileName = c.SVIDKeyFileNameDeprecated
	}

	if c.SVIDBundleFileNameDeprecated != "" {
		if c.SVIDBundleFileName != "" {
			return errors.New("use of svid_bundle_file_name and svidBundleFileName found, use only svid_bundle_file_name")
		}
		log.Warn(getWarning("svidBundleFileName", "svid_bundle_file_name"))
		c.SVIDBundleFileName = c.SVIDBundleFileNameDeprecated
	}

	if c.RenewSignalDeprecated != "" {
		if c.RenewSignal != "" {
			return errors.New("use of renew_signal and renewSignal found, use only renew_signal")
		}
		log.Warn(getWarning("renewSignal", "renew_signal"))
		c.RenewSignal = c.RenewSignalDeprecated
	}

	for _, jwtConfig := range c.JWTSVIDs {
		if jwtConfig.JWTSVIDFilename == "" {
			return errors.New("'jwt_file_name' is required in 'jwt_svids'")
		}
		if jwtConfig.JWTAudience == "" {
			return errors.New("'jwt_audience' is required in 'jwt_svids'")
		}
	}

	if c.AgentAddress == "" {
		c.AgentAddress = os.Getenv("SPIRE_AGENT_ADDRESS")
		if c.AgentAddress == "" {
			c.AgentAddress = defaultAgentAddress
		}
	}

	c.ExitWhenReady = c.ExitWhenReady || exitWhenReady

	x509EmptyCount := countEmpty(c.SVIDFileName, c.SVIDBundleFileName, c.SVIDKeyFileName)
	jwtBundleEmptyCount := countEmpty(c.SVIDBundleFileName)
	if x509EmptyCount == 3 && len(c.JWTSVIDs) == 0 && jwtBundleEmptyCount == 1 {
		return errors.New("at least one of the sets ('svid_file_name', 'svid_key_file_name', 'svid_bundle_file_name'), 'jwt_svids', or 'jwt_bundle_file_name' must be fully specified")
	}

	if x509EmptyCount != 0 && x509EmptyCount != 3 {
		return errors.New("all or none of 'svid_file_name', 'svid_key_file_name', 'svid_bundle_file_name' must be specified")
	}

	return nil
}

func NewSidecarConfig(config *Config, log logrus.FieldLogger) *sidecar.Config {
	sidecarConfig := &sidecar.Config{
		AddIntermediatesToBundle: config.AddIntermediatesToBundle,
		AgentAddress:             config.AgentAddress,
		Cmd:                      config.Cmd,
		CmdArgs:                  config.CmdArgs,
		CertDir:                  config.CertDir,
		ExitWhenReady:            config.ExitWhenReady,
		IncludeFederatedDomains:  config.IncludeFederatedDomains,
		JWTBundleFilename:        config.JWTBundleFilename,
		Log:                      log,
		RenewSignal:              config.RenewSignal,
		SVIDFileName:             config.SVIDFileName,
		SVIDKeyFileName:          config.SVIDKeyFileName,
		SVIDBundleFileName:       config.SVIDBundleFileName,
	}

	for _, jwtSVID := range config.JWTSVIDs {
		sidecarConfig.JWTSVIDs = append(sidecarConfig.JWTSVIDs, sidecar.JWTConfig{
			JWTAudience:     jwtSVID.JWTAudience,
			JWTSVIDFilename: jwtSVID.JWTSVIDFilename,
		})
	}

	return sidecarConfig
}

func getWarning(s1 string, s2 string) string {
	return s1 + " will be deprecated, should be used as " + s2
}

func countEmpty(configs ...string) int {
	cnt := 0
	for _, config := range configs {
		if config == "" {
			cnt++
		}
	}
	return cnt
}
