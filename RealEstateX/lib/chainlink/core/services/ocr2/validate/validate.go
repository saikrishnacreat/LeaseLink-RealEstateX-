package validate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lib/pq"
	"github.com/pelletier/go-toml"
	pkgerrors "github.com/pkg/errors"

	libocr2 "github.com/smartcontractkit/libocr/offchainreporting2plus"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop/reportingplugins"
	"github.com/smartcontractkit/chainlink-common/pkg/types"

	"github.com/smartcontractkit/chainlink/v2/core/config/env"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/ccip/config"
	lloconfig "github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/llo/config"
	mercuryconfig "github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/mercury/config"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/vault"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocrcommon"
	"github.com/smartcontractkit/chainlink/v2/core/services/pipeline"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	evmtypes "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm/types"
	"github.com/smartcontractkit/chainlink/v2/plugins"
)

// ValidatedOracleSpecToml validates an oracle spec that came from TOML
func ValidatedOracleSpecToml(ctx context.Context, config OCR2Config, insConf InsecureConfig, tomlString string, rc plugins.RegistrarConfig) (job.Job, error) {
	var jb = job.Job{}
	var spec job.OCR2OracleSpec
	tree, err := toml.Load(tomlString)
	if err != nil {
		return jb, pkgerrors.Wrap(err, "toml error on load")
	}
	// Note this validates all the fields which implement an UnmarshalText
	// i.e. TransmitterAddress, PeerID...
	err = tree.Unmarshal(&spec)
	if err != nil {
		return jb, pkgerrors.Wrap(err, "toml unmarshal error on spec")
	}
	err = tree.Unmarshal(&jb)
	if err != nil {
		return jb, pkgerrors.Wrap(err, "toml unmarshal error on job")
	}
	jb.OCR2OracleSpec = &spec
	if jb.OCR2OracleSpec.P2PV2Bootstrappers == nil {
		// Empty but non-null, field is non-nullable.
		jb.OCR2OracleSpec.P2PV2Bootstrappers = pq.StringArray{}
	}

	if jb.Type != job.OffchainReporting2 {
		return jb, pkgerrors.Errorf("the only supported type is currently 'offchainreporting2', got %s", jb.Type)
	}
	if _, ok := relay.SupportedNetworks[spec.Relay]; !ok {
		return jb, pkgerrors.Errorf("no such relay %v supported", spec.Relay)
	}
	if len(spec.P2PV2Bootstrappers) > 0 {
		_, err = ocrcommon.ParseBootstrapPeers(spec.P2PV2Bootstrappers)
		if err != nil {
			return jb, err
		}
	}

	if err = validateSpec(ctx, tree, jb, rc); err != nil {
		return jb, err
	}
	if err = validateTimingParameters(config, insConf, spec); err != nil {
		return jb, err
	}
	return jb, nil
}

// Parameters that must be explicitly set by the operator.
var (
	params = map[string]struct{}{
		"type":          {},
		"schemaVersion": {},
		"contractID":    {},
		"relay":         {},
		"relayConfig":   {},
		"pluginType":    {},
	}
	notExpectedParams = map[string]struct{}{
		"isBootstrapPeer":       {},
		"juelsPerFeeCoinSource": {},
	}
)

func validateTimingParameters(ocr2Conf OCR2Config, insConf InsecureConfig, spec job.OCR2OracleSpec) error {
	lc, err := ToLocalConfig(ocr2Conf, insConf, spec)
	if err != nil {
		return err
	}
	return libocr2.SanityCheckLocalConfig(lc)
}

func validateSpec(ctx context.Context, tree *toml.Tree, spec job.Job, rc plugins.RegistrarConfig) error {
	expected, notExpected := ocrcommon.CloneSet(params), ocrcommon.CloneSet(notExpectedParams)
	if err := ocrcommon.ValidateExplicitlySetKeys(tree, expected, notExpected, "ocr2"); err != nil {
		return err
	}

	switch spec.OCR2OracleSpec.PluginType {
	case types.Median:
		if spec.Pipeline.Source == "" {
			return errors.New("no pipeline specified")
		}
	case types.OCR2Keeper:
		return validateOCR2KeeperSpec(spec.OCR2OracleSpec.PluginConfig)
	case types.Functions:
		// TODO validator for DR-OCR spec: https://smartcontract-it.atlassian.net/browse/FUN-112
		return nil
	case types.Mercury:
		return validateOCR2MercurySpec(spec.OCR2OracleSpec, *spec.OCR2OracleSpec.FeedID)
	case types.CCIPExecution:
		return validateOCR2CCIPExecutionSpec(spec.OCR2OracleSpec.PluginConfig)
	case types.CCIPCommit:
		return validateOCR2CCIPCommitSpec(spec.OCR2OracleSpec.PluginConfig)
	case types.LLO:
		return validateOCR2LLOSpec(spec.OCR2OracleSpec.PluginConfig)
	case types.GenericPlugin:
		return validateGenericPluginSpec(ctx, spec.OCR2OracleSpec, rc)
	case types.VaultPlugin:
		return validateVaultPluginSpec(spec.OCR2OracleSpec.PluginConfig)
	case "":
		return errors.New("no plugin specified")
	default:
		return pkgerrors.Errorf("invalid pluginType %s", spec.OCR2OracleSpec.PluginType)
	}

	return nil
}

func validateVaultPluginSpec(jsonConfig job.JSONConfig) error {
	cfg := &vault.Config{}
	err := json.Unmarshal(jsonConfig.Bytes(), cfg)
	if err != nil {
		return fmt.Errorf("failed to validation plugin config: could not unmarshal config: %w", err)
	}

	return cfg.Validate()
}

type PipelineSpec struct {
	Name string `json:"name"`
	Spec string `json:"spec"`
}

type Config struct {
	Pipelines    []PipelineSpec `json:"pipelines"`
	PluginConfig map[string]any `json:"pluginConfig"`
}

type innerConfig struct {
	Command       string            `json:"command"`
	EnvVars       map[string]string `json:"envVars"`
	ProviderType  string            `json:"providerType"`
	PluginName    string            `json:"pluginName"`
	TelemetryType string            `json:"telemetryType"`
	OCRVersion    int               `json:"OCRVersion"`
	Config
}

type OCR2GenericPluginConfig struct {
	innerConfig
}

func (o *OCR2GenericPluginConfig) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &o.innerConfig)
	if err != nil {
		return nil
	}

	m := map[string]any{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	o.PluginConfig = m
	return nil
}

type onchainSigningStrategyInner struct {
	StrategyName string         `json:"strategyName"`
	Config       job.JSONConfig `json:"config"`
}

type OCR2OnchainSigningStrategy struct {
	onchainSigningStrategyInner
}

func (o *OCR2OnchainSigningStrategy) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &o.onchainSigningStrategyInner)
	if err != nil {
		return err
	}

	return nil
}

func (o *OCR2OnchainSigningStrategy) IsMultiChain() bool {
	return o.StrategyName == "multi-chain"
}

func (o *OCR2OnchainSigningStrategy) ConfigCopy() job.JSONConfig {
	copiedConfig := make(job.JSONConfig)
	for k, v := range o.Config {
		copiedConfig[k] = v
	}
	return copiedConfig
}

func (o *OCR2OnchainSigningStrategy) KeyBundleID(name string) (string, error) {
	kbID, ok := o.Config[name]
	if !ok {
		return "", nil
	}
	kbIDString, ok := kbID.(string)
	if !ok {
		return "", fmt.Errorf("expected string %s value, but got: %T", name, kbID)
	}
	return kbIDString, nil
}

func validateGenericPluginSpec(ctx context.Context, spec *job.OCR2OracleSpec, rc plugins.RegistrarConfig) error {
	p := OCR2GenericPluginConfig{}
	err := json.Unmarshal(spec.PluginConfig.Bytes(), &p)
	if err != nil {
		return err
	}

	if p.PluginName == "" {
		return errors.New("generic config invalid: must provide plugin name")
	}

	if p.OCRVersion != 2 && p.OCRVersion != 3 {
		return errors.New("generic config invalid: only OCR version 2 and 3 are supported")
	}

	// OnchainSigningStrategy is optional
	if spec.OnchainSigningStrategy != nil && len(spec.OnchainSigningStrategy.Bytes()) > 0 {
		onchainSigningStrategy := OCR2OnchainSigningStrategy{}
		err = json.Unmarshal(spec.OnchainSigningStrategy.Bytes(), &onchainSigningStrategy)
		if err != nil {
			return err
		}
	}

	plugEnv := env.NewPlugin(p.PluginName)

	command := p.Command
	if command == "" {
		command = plugEnv.Cmd.Get()
	}

	if command == "" {
		return errors.New("generic config invalid: no command found")
	}

	_, err = exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("failed to find binary  %q", command)
	}

	envVars, err := plugins.ParseEnvFile(plugEnv.Env.Get())
	if err != nil {
		return fmt.Errorf("failed to parse env file: %w", err)
	}
	if len(p.EnvVars) > 0 {
		for k, v := range p.EnvVars {
			envVars = append(envVars, k+"="+v)
		}
	}

	loopID := fmt.Sprintf("%s-%s-%s", p.PluginName, spec.ContractID, spec.GetID())
	// Starting and stopping a LOOPP isn't efficient; ideally, we'd initiate the LOOPP once and then reference
	// it later to conserve resources. This code will be revisited once BCF-3126 is implemented, and we have
	// the ability to reference the LOOPP for future use.
	cmdFn, grpcOpts, err := rc.RegisterLOOP(plugins.CmdConfig{
		ID:  loopID,
		Cmd: command,
		Env: envVars,
	})
	if err != nil {
		return fmt.Errorf("failed to register loop: %w", err)
	}
	defer rc.UnregisterLOOP(loopID)

	pluginLggr, _ := logger.New()
	plugin := reportingplugins.NewLOOPPServiceValidation(pluginLggr, grpcOpts, cmdFn)

	err = plugin.Start(ctx)
	if err != nil {
		return err
	}
	defer plugin.Close()

	return plugin.ValidateConfig(ctx, spec.PluginConfig)
}

func validateOCR2KeeperSpec(jsonConfig job.JSONConfig) error {
	return nil
}

func validateOCR2MercurySpec(spec *job.OCR2OracleSpec, feedID [32]byte) error {
	var relayConfig evmtypes.RelayConfig
	err := json.Unmarshal(spec.RelayConfig.Bytes(), &relayConfig)
	if err != nil {
		return pkgerrors.Wrap(err, "error while unmarshalling relay config")
	}

	if len(spec.PluginConfig) == 0 {
		if !relayConfig.EnableTriggerCapability {
			return pkgerrors.Wrap(err, "at least one transmission option must be configured")
		}
		return nil
	}

	var pluginConfig mercuryconfig.PluginConfig
	err = json.Unmarshal(spec.PluginConfig.Bytes(), &pluginConfig)
	if err != nil {
		return pkgerrors.Wrap(err, "error while unmarshalling plugin config")
	}
	return pkgerrors.Wrap(mercuryconfig.ValidatePluginConfig(pluginConfig, feedID), "Mercury PluginConfig is invalid")
}

func validateOCR2CCIPExecutionSpec(jsonConfig job.JSONConfig) error {
	if jsonConfig == nil {
		return errors.New("pluginConfig is empty")
	}
	var cfg config.ExecPluginJobSpecConfig
	err := json.Unmarshal(jsonConfig.Bytes(), &cfg)
	if err != nil {
		return pkgerrors.Wrap(err, "error while unmarshalling plugin config")
	}
	if cfg.USDCConfig != (config.USDCConfig{}) {
		return cfg.USDCConfig.ValidateUSDCConfig()
	}
	return nil
}

func validateOCR2CCIPCommitSpec(jsonConfig job.JSONConfig) error {
	if jsonConfig == nil {
		return errors.New("pluginConfig is empty")
	}
	var cfg config.CommitPluginJobSpecConfig
	err := json.Unmarshal(jsonConfig.Bytes(), &cfg)
	if err != nil {
		return pkgerrors.Wrap(err, "error while unmarshalling plugin config")
	}

	// Ensure that either the tokenPricesUSDPipeline or the priceGetterConfig is set, but not both.
	emptyPipeline := strings.Trim(cfg.TokenPricesUSDPipeline, "\n\t ") == ""
	emptyPriceGetter := cfg.PriceGetterConfig == nil
	if emptyPipeline && emptyPriceGetter {
		return errors.New("either tokenPricesUSDPipeline or priceGetterConfig must be set")
	}
	if !emptyPipeline && !emptyPriceGetter {
		return fmt.Errorf("only one of tokenPricesUSDPipeline or priceGetterConfig must be set: %s and %v", cfg.TokenPricesUSDPipeline, cfg.PriceGetterConfig)
	}

	if !emptyPipeline {
		_, err = pipeline.Parse(cfg.TokenPricesUSDPipeline)
		if err != nil {
			return pkgerrors.Wrap(err, "invalid token prices pipeline")
		}
	} else {
		// Validate prices config (like it was done for the pipeline).
		if emptyPriceGetter {
			return pkgerrors.New("priceGetterConfig is empty")
		}
	}

	return nil
}

func validateOCR2LLOSpec(jsonConfig job.JSONConfig) error {
	var pluginConfig lloconfig.PluginConfig
	err := json.Unmarshal(jsonConfig.Bytes(), &pluginConfig)
	if err != nil {
		return pkgerrors.Wrap(err, "error while unmarshaling plugin config")
	}
	return pkgerrors.Wrap(pluginConfig.Validate(), "LLO PluginConfig is invalid")
}
