package integrationtesthelpers

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	types3 "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"

	"github.com/smartcontractkit/freeport"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/confighelper"
	types4 "github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	cciptypes "github.com/smartcontractkit/chainlink-common/pkg/types/ccip"
	"github.com/smartcontractkit/chainlink-evm/pkg/chains/legacyevm"
	pb "github.com/smartcontractkit/chainlink-protos/orchestrator/feedsmanager"

	"github.com/smartcontractkit/chainlink-evm/pkg/client"
	"github.com/smartcontractkit/chainlink-evm/pkg/config/toml"
	"github.com/smartcontractkit/chainlink-evm/pkg/logpoller"
	"github.com/smartcontractkit/chainlink-evm/pkg/utils"
	evmUtils "github.com/smartcontractkit/chainlink-evm/pkg/utils/big"

	price_registry_1_2_0 "github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_2_0/price_registry"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_5_0/commit_store"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_5_0/evm_2_evm_offramp"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_5_0/evm_2_evm_onramp"
	configv2 "github.com/smartcontractkit/chainlink/v2/core/config/toml"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/logger/audit"
	"github.com/smartcontractkit/chainlink/v2/core/services/chainlink"
	feeds2 "github.com/smartcontractkit/chainlink/v2/core/services/feeds"
	feedsMocks "github.com/smartcontractkit/chainlink/v2/core/services/feeds/mocks"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/chaintype"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/csakey"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/ocr2key"
	ksMocks "github.com/smartcontractkit/chainlink/v2/core/services/keystore/mocks"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/ccip/abihelpers"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/ccip/internal/ccipdata"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/ccip/internal/ccipdata/v1_2_0"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/ccip/internal/ccipdata/v1_5_0"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/plugins/ccip/testhelpers"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocr2/validate"
	"github.com/smartcontractkit/chainlink/v2/core/services/ocrbootstrap"
	evmrelay "github.com/smartcontractkit/chainlink/v2/core/services/relay/evm"
	clutils "github.com/smartcontractkit/chainlink/v2/core/utils"
	"github.com/smartcontractkit/chainlink/v2/core/utils/crypto"
	"github.com/smartcontractkit/chainlink/v2/core/utils/testutils/heavyweight"
)

const (
	execSpecTemplate = `
		type = "offchainreporting2"
		schemaVersion = 1
		name = "ccip-exec-1"
		externalJobID      = "67ffad71-d90f-4fe3-b4e4-494924b707fb"
		forwardingAllowed = false
		maxTaskDuration = "0s"
		contractID = "%s"
		contractConfigConfirmations = 1
		contractConfigTrackerPollInterval = "20s"
		ocrKeyBundleID = "%s"
		relay = "evm"
		pluginType = "ccip-execution"
		transmitterID = "%s"

		[relayConfig]
		chainID = 1_337

		[pluginConfig]
		destStartBlock = 50

	    [pluginConfig.USDCConfig]
	    AttestationAPI = "http://blah.com"
	    SourceMessageTransmitterAddress = "%s"
	    SourceTokenAddress = "%s"
		AttestationAPITimeoutSeconds = 10
	`
	commitSpecTemplatePipeline = `
		type = "offchainreporting2"
		schemaVersion = 1
		name = "ccip-commit-1"
		externalJobID = "13c997cf-1a14-4ab7-9068-07ee6d2afa55"
		forwardingAllowed = false
		maxTaskDuration = "0s"
		contractID = "%s"
		contractConfigConfirmations = 1
		contractConfigTrackerPollInterval = "20s"
		ocrKeyBundleID = "%s"
		relay = "evm"
		pluginType = "ccip-commit"
		transmitterID = "%s"

		[relayConfig]
		chainID = 1_337

		[pluginConfig]
		destStartBlock = 50
		offRamp = "%s"
		tokenPricesUSDPipeline = """
		%s
		"""
	`
	commitSpecTemplateDynamicPriceGetter = `
		type = "offchainreporting2"
		schemaVersion = 1
		name = "ccip-commit-1"
		externalJobID = "13c997cf-1a14-4ab7-9068-07ee6d2afa55"
		forwardingAllowed = false
		maxTaskDuration = "0s"
		contractID = "%s"
		contractConfigConfirmations = 1
		contractConfigTrackerPollInterval = "20s"
		ocrKeyBundleID = "%s"
		relay = "evm"
		pluginType = "ccip-commit"
		transmitterID = "%s"

		[relayConfig]
		chainID = 1_337

		[pluginConfig]
		destStartBlock = 50
		offRamp = "%s"
		priceGetterConfig = """
		%s
		"""
	`
)

type Node struct {
	App             chainlink.Application
	Transmitter     common.Address
	PaymentReceiver common.Address
	KeyBundle       ocr2key.KeyBundle
}

func (node *Node) FindJobIDForContract(t *testing.T, addr common.Address) int32 {
	jobs := node.App.JobSpawner().ActiveJobs()
	for _, j := range jobs {
		if j.Type == job.OffchainReporting2 && j.OCR2OracleSpec.ContractID == addr.Hex() {
			return j.ID
		}
	}
	t.Fatalf("Could not find job for contract %s", addr.Hex())
	return 0
}

func (node *Node) EventuallyNodeUsesUpdatedPriceRegistry(t *testing.T, ccipContracts CCIPIntegrationTestHarness) logpoller.Log {
	cs, err := node.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(ccipContracts.Dest.ChainID, 10))
	require.NoError(t, err)
	c, ok := cs.(legacyevm.Chain)
	require.True(t, ok)
	var log logpoller.Log
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		ccipContracts.Source.Chain.Commit()
		ccipContracts.Dest.Chain.Commit()
		log, err := c.LogPoller().LatestLogByEventSigWithConfs(
			testutils.Context(t),
			v1_2_0.UsdPerUnitGasUpdated,
			ccipContracts.Dest.PriceRegistry.Address(),
			0,
		)
		// err can be transient errors such as sql row set empty
		if err != nil {
			return false
		}
		return log != nil
	}, testutils.WaitTimeout(t), 1*time.Second).Should(gomega.BeTrue(), "node is not using updated price registry %s", ccipContracts.Dest.PriceRegistry.Address().Hex())
	return log
}

func (node *Node) EventuallyNodeUsesNewCommitConfig(t *testing.T, ccipContracts CCIPIntegrationTestHarness, commitCfg ccipdata.CommitOnchainConfig) logpoller.Log {
	cs, err := node.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(ccipContracts.Dest.ChainID, 10))
	require.NoError(t, err)
	c, ok := cs.(legacyevm.Chain)
	require.True(t, ok)
	var log logpoller.Log
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		ccipContracts.Source.Chain.Commit()
		ccipContracts.Dest.Chain.Commit()
		log, err := c.LogPoller().LatestLogByEventSigWithConfs(
			testutils.Context(t),
			evmrelay.OCR2AggregatorLogDecoder.EventSig(),
			ccipContracts.Dest.CommitStore.Address(),
			0,
		)
		require.NoError(t, err)
		var latestCfg ccipdata.CommitOnchainConfig
		if log != nil {
			latestCfg, err = DecodeCommitOnChainConfig(log.Data)
			require.NoError(t, err)
			return latestCfg == commitCfg
		}
		return false
	}, testutils.WaitTimeout(t), 1*time.Second).Should(gomega.BeTrue(), "node is using old cfg")
	return log
}

func (node *Node) EventuallyNodeUsesNewExecConfig(t *testing.T, ccipContracts CCIPIntegrationTestHarness, execCfg v1_5_0.ExecOnchainConfig) logpoller.Log {
	cs, err := node.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(ccipContracts.Dest.ChainID, 10))
	require.NoError(t, err)
	c, ok := cs.(legacyevm.Chain)
	require.True(t, ok)
	var log logpoller.Log
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		ccipContracts.Source.Chain.Commit()
		ccipContracts.Dest.Chain.Commit()
		log, err := c.LogPoller().LatestLogByEventSigWithConfs(
			testutils.Context(t),
			evmrelay.OCR2AggregatorLogDecoder.EventSig(),
			ccipContracts.Dest.OffRamp.Address(),
			0,
		)
		require.NoError(t, err)
		var latestCfg v1_5_0.ExecOnchainConfig
		if log != nil {
			latestCfg, err = DecodeExecOnChainConfig(log.Data)
			require.NoError(t, err)
			return latestCfg == execCfg
		}
		return false
	}, testutils.WaitTimeout(t), 1*time.Second).Should(gomega.BeTrue(), "node is using old cfg")
	return log
}

func (node *Node) EventuallyHasReqSeqNum(t *testing.T, ccipContracts *CCIPIntegrationTestHarness, onRamp common.Address, seqNum int) logpoller.Log {
	cs, err := node.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(ccipContracts.Source.ChainID, 10))
	require.NoError(t, err)
	c, ok := cs.(legacyevm.Chain)
	require.True(t, ok)
	var log logpoller.Log
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		ccipContracts.Source.Chain.Commit()
		ccipContracts.Dest.Chain.Commit()
		lgs, err := c.LogPoller().LogsDataWordRange(
			testutils.Context(t),
			v1_2_0.CCIPSendRequestEventSig,
			onRamp,
			v1_2_0.CCIPSendRequestSeqNumIndex,
			abihelpers.EvmWord(uint64(seqNum)),
			abihelpers.EvmWord(uint64(seqNum)),
			1,
		)
		require.NoError(t, err)
		t.Log("Send requested", len(lgs))
		if len(lgs) == 1 {
			log = lgs[0]
			return true
		}
		return false
	}, testutils.WaitTimeout(t), 1*time.Second).Should(gomega.BeTrue(), "eventually has seq num")
	return log
}

func (node *Node) EventuallyHasExecutedSeqNums(t *testing.T, ccipContracts *CCIPIntegrationTestHarness, offRamp common.Address, minSeqNum int, maxSeqNum int) []logpoller.Log {
	cs, err := node.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(ccipContracts.Dest.ChainID, 10))
	require.NoError(t, err)
	c, ok := cs.(legacyevm.Chain)
	require.True(t, ok)
	var logs []logpoller.Log
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		ccipContracts.Source.Chain.Commit()
		ccipContracts.Dest.Chain.Commit()
		lgs, err := c.LogPoller().IndexedLogsTopicRange(
			testutils.Context(t),
			v1_2_0.ExecutionStateChangedEvent,
			offRamp,
			v1_2_0.ExecutionStateChangedSeqNrIndex,
			abihelpers.EvmWord(uint64(minSeqNum)),
			abihelpers.EvmWord(uint64(maxSeqNum)),
			1,
		)
		require.NoError(t, err)
		t.Logf("Have executed logs %d want %d", len(lgs), maxSeqNum-minSeqNum+1)
		if len(lgs) == maxSeqNum-minSeqNum+1 {
			logs = lgs
			t.Logf("Seq Num %d-%d executed", minSeqNum, maxSeqNum)
			return true
		}
		return false
	}, testutils.WaitTimeout(t), 1*time.Second).Should(gomega.BeTrue(), "eventually has not executed seq num")
	return logs
}

func (node *Node) ConsistentlySeqNumHasNotBeenExecuted(t *testing.T, ccipContracts *CCIPIntegrationTestHarness, offRamp common.Address, seqNum int) logpoller.Log {
	cs, err := node.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(ccipContracts.Dest.ChainID, 10))
	require.NoError(t, err)
	c, ok := cs.(legacyevm.Chain)
	require.True(t, ok)
	var log logpoller.Log
	gomega.NewGomegaWithT(t).Consistently(func() bool {
		ccipContracts.Source.Chain.Commit()
		ccipContracts.Dest.Chain.Commit()
		lgs, err := c.LogPoller().IndexedLogsTopicRange(
			testutils.Context(t),
			v1_2_0.ExecutionStateChangedEvent,
			offRamp,
			v1_2_0.ExecutionStateChangedSeqNrIndex,
			abihelpers.EvmWord(uint64(seqNum)),
			abihelpers.EvmWord(uint64(seqNum)),
			1,
		)
		require.NoError(t, err)
		t.Log("Executed logs", lgs)
		if len(lgs) == 1 {
			log = lgs[0]
			return true
		}
		return false
	}, 10*time.Second, 1*time.Second).Should(gomega.BeFalse(), "seq number got executed")
	return log
}

func (node *Node) AddJob(t *testing.T, spec *OCR2TaskJobSpec) {
	specString, err := spec.String()
	require.NoError(t, err)
	ccipJob, err := validate.ValidatedOracleSpecToml(
		testutils.Context(t),
		node.App.GetConfig().OCR2(),
		node.App.GetConfig().Insecure(),
		specString,
		// FIXME Ani
		nil,
	)
	require.NoError(t, err)
	err = node.App.AddJobV2(t.Context(), &ccipJob)
	require.NoError(t, err)
}

func (node *Node) AddBootstrapJob(t *testing.T, spec *OCR2TaskJobSpec) {
	specString, err := spec.String()
	require.NoError(t, err)
	ccipJob, err := ocrbootstrap.ValidatedBootstrapSpecToml(specString)
	require.NoError(t, err)
	err = node.App.AddJobV2(t.Context(), &ccipJob)
	require.NoError(t, err)
}

func (node *Node) AddJobsWithSpec(t *testing.T, jobSpec *OCR2TaskJobSpec) {
	// set node specific values
	jobSpec.OCR2OracleSpec.OCRKeyBundleID.SetValid(node.KeyBundle.ID())
	jobSpec.OCR2OracleSpec.TransmitterID.SetValid(node.Transmitter.Hex())
	node.AddJob(t, jobSpec)
}

func setupNodeCCIP(
	t *testing.T,
	owner *bind.TransactOpts,
	port int64,
	dbName string,
	sourceChain *testhelpers.Backend, destChain *testhelpers.Backend,
	sourceChainID *big.Int, destChainID *big.Int,
	bootstrapPeerID string,
	bootstrapPort int64,
	sourceFinalityDepth, destFinalityDepth uint32,
) (chainlink.Application, string, common.Address, ocr2key.KeyBundle) {
	trueRef, falseRef := true, false

	// Do not want to load fixtures as they contain a dummy chainID.
	loglevel := configv2.LogLevel(zap.DebugLevel)
	config, db := heavyweight.FullTestDBNoFixturesV2(t, func(c *chainlink.Config, _ *chainlink.Secrets) {
		p2pAddresses := []string{
			fmt.Sprintf("127.0.0.1:%d", port),
		}
		c.Log.Level = &loglevel
		c.Feature.CCIP = &trueRef
		c.Feature.UICSAKeys = &trueRef
		c.Feature.FeedsManager = &trueRef
		c.OCR.Enabled = &falseRef
		c.OCR.DefaultTransactionQueueDepth = ptr.To[uint32](200)
		c.OCR2.Enabled = &trueRef
		c.Feature.LogPoller = &trueRef
		c.P2P.V2.Enabled = &trueRef

		dur, err := config.NewDuration(500 * time.Millisecond)
		if err != nil {
			panic(err)
		}
		c.P2P.V2.DeltaDial = &dur

		dur2, err := config.NewDuration(5 * time.Second)
		if err != nil {
			panic(err)
		}

		c.P2P.V2.DeltaReconcile = &dur2
		c.P2P.V2.ListenAddresses = &p2pAddresses
		c.P2P.V2.AnnounceAddresses = &p2pAddresses

		c.EVM = []*toml.EVMConfig{createConfigV2Chain(sourceChainID, sourceFinalityDepth), createConfigV2Chain(destChainID, destFinalityDepth)}

		if bootstrapPeerID != "" {
			// Supply the bootstrap IP and port as a V2 peer address
			c.P2P.V2.DefaultBootstrappers = &[]commontypes.BootstrapperLocator{
				{
					PeerID: bootstrapPeerID, Addrs: []string{
						fmt.Sprintf("127.0.0.1:%d", bootstrapPort),
					},
				},
			}
		}
	})

	lggr := logger.TestLogger(t)
	ctx := testutils.Context(t)

	// The in-memory geth sim does not let you create a custom ChainID, it will always be 1337.
	// In particular this means that if you sign an eip155 tx, the chainID used MUST be 1337
	// and the CHAINID op code will always emit 1337. To work around this to simulate a "multichain"
	// test, we fake different chainIDs using the wrapped sim cltest.SimulatedBackend so the RPC
	// appears to operate on different chainIDs and we use an EthKeyStoreSim wrapper which always
	// signs 1337 see https://github.com/smartcontractkit/chainlink-ccip/blob/a24dd436810250a458d27d8bb3fb78096afeb79c/core/services/ocr2/plugins/ccip/testhelpers/simulated_backend.go#L35
	sourceClient := client.NewSimulatedBackendClient(t, sourceChain, sourceChainID)
	destClient := client.NewSimulatedBackendClient(t, destChain, destChainID)
	csaKeyStore := ksMocks.NewCSA(t)

	key, err := csakey.NewV2()
	require.NoError(t, err)
	csaKeyStore.On("EnsureKey", mock.Anything).Return(nil)
	csaKeyStore.On("GetAll").Return([]csakey.KeyV2{key}, nil)
	keyStore := NewKsa(db, lggr, csaKeyStore)

	app, err := chainlink.NewApplication(ctx, chainlink.ApplicationOpts{
		Config:   config,
		DS:       db,
		KeyStore: keyStore,
		EVMFactoryConfigFn: func(fc *chainlink.EVMFactoryConfig) {
			fc.GenEthClient = func(chainID *big.Int) client.Client {
				if chainID.String() == sourceChainID.String() {
					return sourceClient
				} else if chainID.String() == destChainID.String() {
					return destClient
				}
				t.Fatalf("invalid chain ID %v", chainID.String())
				return nil
			}
		},
		Logger:                   lggr,
		ExternalInitiatorManager: nil,
		CloseLogger:              lggr.Sync,
		UnrestrictedHTTPClient:   &http.Client{},
		RestrictedHTTPClient:     &http.Client{},
		AuditLogger:              audit.NoopLogger,
	})
	require.NoError(t, err)
	require.NoError(t, app.GetKeyStore().Unlock(ctx, "password"))
	_, err = app.GetKeyStore().P2P().Create(ctx)
	require.NoError(t, err)

	p2pIDs, err := app.GetKeyStore().P2P().GetAll()
	require.NoError(t, err)
	require.Len(t, p2pIDs, 1)
	peerID := p2pIDs[0].PeerID()

	testCtx := testutils.Context(t)
	_, err = app.GetKeyStore().Eth().Create(testCtx, destChainID)
	require.NoError(t, err)
	sendingKeys, err := app.GetKeyStore().Eth().EnabledKeysForChain(testCtx, destChainID)
	require.NoError(t, err)
	require.Len(t, sendingKeys, 1)
	transmitter := sendingKeys[0].Address
	s, err := app.GetKeyStore().Eth().GetState(testCtx, sendingKeys[0].ID(), destChainID)
	require.NoError(t, err)
	lggr.Debug(fmt.Sprintf("Transmitter address %s chainID %s", transmitter, s.EVMChainID.String()))

	// Fund the commitTransmitter address with some ETH
	n, err := destChain.Client().NonceAt(t.Context(), owner.From, nil)
	require.NoError(t, err)

	tx := types3.NewTransaction(n, transmitter, big.NewInt(1000000000000000000), 21000, big.NewInt(1000000000), nil)
	signedTx, err := owner.Signer(owner.From, tx)
	require.NoError(t, err)
	err = destChain.Client().SendTransaction(t.Context(), signedTx)
	require.NoError(t, err)
	destChain.Commit()

	kb, err := app.GetKeyStore().OCR2().Create(ctx, chaintype.EVM)
	require.NoError(t, err)
	return app, peerID.Raw(), transmitter, kb
}

func createConfigV2Chain(chainID *big.Int, finalityDepth uint32) *toml.EVMConfig {
	// NOTE: For the executor jobs, the default of 500k is insufficient for a 3 message batch
	defaultGasLimit := uint64(5000000)
	tr := true

	sourceC := toml.Defaults((*evmUtils.Big)(chainID))
	sourceC.GasEstimator.LimitDefault = &defaultGasLimit
	fixedPrice := "FixedPrice"
	sourceC.GasEstimator.Mode = &fixedPrice
	d, _ := config.NewDuration(100 * time.Millisecond)
	sourceC.LogPollInterval = &d
	sourceC.FinalityDepth = &finalityDepth
	return &toml.EVMConfig{
		ChainID: (*evmUtils.Big)(chainID),
		Enabled: &tr,
		Chain:   sourceC,
		Nodes:   toml.EVMNodes{&toml.Node{}},
	}
}

type CCIPIntegrationTestHarness struct {
	testhelpers.CCIPContracts
	Nodes     []Node
	Bootstrap Node
}

func SetupCCIPIntegrationTH(t *testing.T, sourceChainID, sourceChainSelector, destChainID, destChainSelector uint64,
	sourceFinalityDepth, destFinalityDepth uint32) CCIPIntegrationTestHarness {
	return CCIPIntegrationTestHarness{
		CCIPContracts: testhelpers.SetupCCIPContracts(t, sourceChainID, sourceChainSelector, destChainID,
			destChainSelector, sourceFinalityDepth, destFinalityDepth),
	}
}

func (c *CCIPIntegrationTestHarness) CreatePricesPipeline(t *testing.T) (string, *httptest.Server, *httptest.Server) {
	linkUSD := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"UsdPerLink": "8000000000000000000"}`))
		require.NoError(t, err)
	}))
	t.Cleanup(linkUSD.Close)

	ethUSD := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"UsdPerETH": "1700000000000000000000"}`))
		require.NoError(t, err)
	}))
	t.Cleanup(ethUSD.Close)

	sourceWrappedNative, err := c.Source.Router.GetWrappedNative(nil)
	require.NoError(t, err)
	destWrappedNative, err := c.Dest.Router.GetWrappedNative(nil)
	require.NoError(t, err)
	tokenPricesUSDPipeline := fmt.Sprintf(`
// Price 1
link [type=http method=GET url="%s"];
link_parse [type=jsonparse path="UsdPerLink"];
link->link_parse;
eth [type=http method=GET url="%s"];
eth_parse [type=jsonparse path="UsdPerETH"];
eth->eth_parse;
merge [type=merge left="{}" right="{\\\"%s\\\":$(link_parse), \\\"%s\\\":$(eth_parse), \\\"%s\\\":$(eth_parse)}"];`,
		linkUSD.URL, ethUSD.URL, c.Dest.LinkToken.Address(), sourceWrappedNative, destWrappedNative)

	return tokenPricesUSDPipeline, linkUSD, ethUSD
}

func (c *CCIPIntegrationTestHarness) AddAllJobs(t *testing.T, jobParams CCIPJobSpecParams) {
	jobParams.OffRamp = c.Dest.OffRamp.Address()

	commitSpec, err := jobParams.CommitJobSpec()
	require.NoError(t, err)
	geExecutionSpec, err := jobParams.ExecutionJobSpec()
	require.NoError(t, err)
	nodes := c.Nodes
	for _, node := range nodes {
		node.AddJobsWithSpec(t, commitSpec)
		node.AddJobsWithSpec(t, geExecutionSpec)
	}
}

func (c *CCIPIntegrationTestHarness) jobSpecProposal(t *testing.T, specTemplate string, f func() (*OCR2TaskJobSpec, error), feedsManagerId int64, version int32, opts ...any) feeds2.ProposeJobArgs {
	spec, err := f()
	require.NoError(t, err)

	args := []any{spec.OCR2OracleSpec.ContractID}
	args = append(args, opts...)

	return feeds2.ProposeJobArgs{
		FeedsManagerID: feedsManagerId,
		RemoteUUID:     uuid.New(),
		Multiaddrs:     nil,
		Version:        version,
		Spec:           fmt.Sprintf(specTemplate, args...),
	}
}

func (c *CCIPIntegrationTestHarness) SetupFeedsManager(t *testing.T) {
	ctx := testutils.Context(t)
	for _, node := range c.Nodes {
		f := node.App.GetFeedsService()

		managers, err := f.ListManagers(ctx)
		require.NoError(t, err)
		if len(managers) > 0 {
			// Use at most one feeds manager, don't register if one already exists
			continue
		}

		secret := utils.RandomBytes32()
		pkey, err := crypto.PublicKeyFromHex(hex.EncodeToString(secret[:]))
		require.NoError(t, err)

		m := feeds2.RegisterManagerParams{
			Name:      "CCIP",
			URI:       "http://localhost:8080",
			PublicKey: *pkey,
		}

		connManager := feedsMocks.NewConnectionsManager(t)
		connManager.On("Connect", mock.Anything).Maybe()
		connManager.On("GetClient", mock.Anything).Maybe().Return(NoopFeedsClient{}, nil)
		connManager.On("Close").Maybe().Return()
		connManager.On("IsConnected", mock.Anything).Maybe().Return(true)
		f.Unsafe_SetConnectionsManager(connManager)

		_, err = f.RegisterManager(testutils.Context(t), m)
		require.NoError(t, err)
	}
}

func (c *CCIPIntegrationTestHarness) ApproveJobSpecs(t *testing.T, jobParams CCIPJobSpecParams) {
	ctx := testutils.Context(t)

	for _, node := range c.Nodes {
		f := node.App.GetFeedsService()
		managers, err := f.ListManagers(ctx)
		require.NoError(t, err)
		require.Len(t, managers, 1, "expected exactly one feeds manager")

		execSpec := c.jobSpecProposal(
			t,
			execSpecTemplate,
			jobParams.ExecutionJobSpec,
			managers[0].ID,
			1,
			node.KeyBundle.ID(),
			node.Transmitter.Hex(),
			utils.RandomAddress().String(),
			utils.RandomAddress().String(),
		)
		execId, err := f.ProposeJob(ctx, &execSpec)
		require.NoError(t, err)

		err = f.ApproveSpec(ctx, execId, true)
		require.NoError(t, err)

		var commitSpec feeds2.ProposeJobArgs
		if jobParams.TokenPricesUSDPipeline != "" {
			commitSpec = c.jobSpecProposal(
				t,
				commitSpecTemplatePipeline,
				jobParams.CommitJobSpec,
				managers[0].ID,
				2,
				node.KeyBundle.ID(),
				node.Transmitter.Hex(),
				jobParams.OffRamp.String(),
				jobParams.TokenPricesUSDPipeline,
			)
		} else {
			commitSpec = c.jobSpecProposal(
				t,
				commitSpecTemplateDynamicPriceGetter,
				jobParams.CommitJobSpec,
				managers[0].ID,
				2,
				node.KeyBundle.ID(),
				node.Transmitter.Hex(),
				jobParams.OffRamp.String(),
				jobParams.PriceGetterConfig,
			)
		}

		commitId, err := f.ProposeJob(ctx, &commitSpec)
		require.NoError(t, err)

		err = f.ApproveSpec(ctx, commitId, true)
		require.NoError(t, err)
	}
}

func (c *CCIPIntegrationTestHarness) AllNodesHaveReqSeqNum(t *testing.T, seqNum int, onRampOpts ...common.Address) logpoller.Log {
	var log logpoller.Log
	nodes := c.Nodes
	var onRamp common.Address
	if len(onRampOpts) > 0 {
		onRamp = onRampOpts[0]
	} else {
		require.NotNil(t, c.Source.OnRamp, "no onramp configured")
		onRamp = c.Source.OnRamp.Address()
	}
	for _, node := range nodes {
		log = node.EventuallyHasReqSeqNum(t, c, onRamp, seqNum)
	}
	return log
}

func (c *CCIPIntegrationTestHarness) AllNodesHaveExecutedSeqNums(t *testing.T, minSeqNum int, maxSeqNum int, offRampOpts ...common.Address) []logpoller.Log {
	var logs []logpoller.Log
	nodes := c.Nodes
	var offRamp common.Address

	if len(offRampOpts) > 0 {
		offRamp = offRampOpts[0]
	} else {
		require.NotNil(t, c.Dest.OffRamp, "no offramp configured")
		offRamp = c.Dest.OffRamp.Address()
	}
	for _, node := range nodes {
		logs = node.EventuallyHasExecutedSeqNums(t, c, offRamp, minSeqNum, maxSeqNum)
	}
	return logs
}

func (c *CCIPIntegrationTestHarness) NoNodesHaveExecutedSeqNum(t *testing.T, seqNum int, offRampOpts ...common.Address) logpoller.Log {
	var log logpoller.Log
	nodes := c.Nodes
	var offRamp common.Address
	if len(offRampOpts) > 0 {
		offRamp = offRampOpts[0]
	} else {
		require.NotNil(t, c.Dest.OffRamp, "no offramp configured")
		offRamp = c.Dest.OffRamp.Address()
	}
	for _, node := range nodes {
		log = node.ConsistentlySeqNumHasNotBeenExecuted(t, c, offRamp, seqNum)
	}
	return log
}

func (c *CCIPIntegrationTestHarness) EventuallyPriceRegistryUpdated(t *testing.T, block uint64, srcSelector uint64, tokens []common.Address, sourceNative common.Address, priceRegistryOpts ...common.Address) {
	var priceRegistry *price_registry_1_2_0.PriceRegistry
	var err error
	if len(priceRegistryOpts) > 0 {
		priceRegistry, err = price_registry_1_2_0.NewPriceRegistry(priceRegistryOpts[0], c.Dest.Chain.Client())
		require.NoError(t, err)
	} else {
		require.NotNil(t, c.Dest.PriceRegistry, "no priceRegistry configured")
		priceRegistry = c.Dest.PriceRegistry
	}

	g := gomega.NewGomegaWithT(t)
	g.Eventually(func() bool {
		it, err := priceRegistry.FilterUsdPerTokenUpdated(&bind.FilterOpts{Start: block}, tokens)
		g.Expect(err).NotTo(gomega.HaveOccurred(), "Error filtering UsdPerTokenUpdated event")

		tokensFetched := make([]common.Address, 0, len(tokens))
		for it.Next() {
			tokenFetched := it.Event.Token
			tokensFetched = append(tokensFetched, tokenFetched)
			t.Log("Token price updated", tokenFetched.String(), it.Event.Value.String(), it.Event.Timestamp.String())
		}

		for _, token := range tokens {
			if !slices.Contains(tokensFetched, token) {
				return false
			}
		}

		return true
	}, testutils.WaitTimeout(t), 10*time.Second).Should(gomega.BeTrue(), "Tokens prices has not been updated")

	g.Eventually(func() bool {
		it, err := priceRegistry.FilterUsdPerUnitGasUpdated(&bind.FilterOpts{Start: block}, []uint64{srcSelector})
		g.Expect(err).NotTo(gomega.HaveOccurred(), "Error filtering UsdPerUnitGasUpdated event")
		g.Expect(it.Next()).To(gomega.BeTrue(), "No UsdPerUnitGasUpdated event found")

		return true
	}, testutils.WaitTimeout(t), 10*time.Second).Should(gomega.BeTrue(), "source gas price has not been updated")
}

func (c *CCIPIntegrationTestHarness) EventuallyCommitReportAccepted(t *testing.T, currentBlock uint64, commitStoreOpts ...common.Address) commit_store.CommitStoreCommitReport {
	var commitStore *commit_store.CommitStore
	var err error
	if len(commitStoreOpts) > 0 {
		commitStore, err = commit_store.NewCommitStore(commitStoreOpts[0], c.Dest.Chain.Client())
		require.NoError(t, err)
	} else {
		require.NotNil(t, c.Dest.CommitStore, "no commitStore configured")
		commitStore = c.Dest.CommitStore
	}
	g := gomega.NewGomegaWithT(t)
	var report commit_store.CommitStoreCommitReport
	g.Eventually(func() bool {
		it, err := commitStore.FilterReportAccepted(&bind.FilterOpts{Start: currentBlock})
		g.Expect(err).NotTo(gomega.HaveOccurred(), "Error filtering ReportAccepted event")
		g.Expect(it.Next()).To(gomega.BeTrue(), "No ReportAccepted event found")
		report = it.Event.Report
		if report.MerkleRoot != [32]byte{} {
			t.Log("Report Accepted by commitStore")
			return true
		}
		return false
	}, testutils.WaitTimeout(t), 1*time.Second).Should(gomega.BeTrue(), "report has not been committed")
	return report
}

func (c *CCIPIntegrationTestHarness) EventuallyExecutionStateChangedToSuccess(t *testing.T, seqNum []uint64, blockNum uint64, offRampOpts ...common.Address) {
	var offRamp *evm_2_evm_offramp.EVM2EVMOffRamp
	var err error
	if len(offRampOpts) > 0 {
		offRamp, err = evm_2_evm_offramp.NewEVM2EVMOffRamp(offRampOpts[0], c.Dest.Chain.Client())
		require.NoError(t, err)
	} else {
		require.NotNil(t, c.Dest.OffRamp, "no offRamp configured")
		offRamp = c.Dest.OffRamp
	}
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		it, err := offRamp.FilterExecutionStateChanged(&bind.FilterOpts{Start: blockNum}, seqNum, [][32]byte{})
		require.NoError(t, err)
		for it.Next() {
			if cciptypes.MessageExecutionState(it.Event.State) == cciptypes.ExecutionStateSuccess {
				t.Logf("ExecutionStateChanged event found for seqNum %d", it.Event.SequenceNumber)
				return true
			}
		}
		c.Source.Chain.Commit()
		c.Dest.Chain.Commit()
		return false
	}, testutils.WaitTimeout(t), time.Second).
		Should(gomega.BeTrue(), "ExecutionStateChanged Event")
}

func (c *CCIPIntegrationTestHarness) EventuallyReportCommitted(t *testing.T, max int, commitStoreOpts ...common.Address) uint64 {
	var commitStore *commit_store.CommitStore
	var err error
	var committedSeqNum uint64
	if len(commitStoreOpts) > 0 {
		commitStore, err = commit_store.NewCommitStore(commitStoreOpts[0], c.Dest.Chain.Client())
		require.NoError(t, err)
	} else {
		require.NotNil(t, c.Dest.CommitStore, "no commitStore configured")
		commitStore = c.Dest.CommitStore
	}
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		minSeqNum, err := commitStore.GetExpectedNextSequenceNumber(nil)
		require.NoError(t, err)
		c.Source.Chain.Commit()
		c.Dest.Chain.Commit()
		t.Log("next expected seq num reported", minSeqNum)
		committedSeqNum = minSeqNum
		return minSeqNum > uint64(max)
	}, testutils.WaitTimeout(t), time.Second).Should(gomega.BeTrue(), "report has not been committed")
	return committedSeqNum
}

func (c *CCIPIntegrationTestHarness) EventuallySendRequested(t *testing.T, seqNum uint64, onRampOpts ...common.Address) {
	var onRamp *evm_2_evm_onramp.EVM2EVMOnRamp
	var err error
	if len(onRampOpts) > 0 {
		onRamp, err = evm_2_evm_onramp.NewEVM2EVMOnRamp(onRampOpts[0], c.Source.Chain.Client())
		require.NoError(t, err)
	} else {
		require.NotNil(t, c.Source.OnRamp, "no onRamp configured")
		onRamp = c.Source.OnRamp
	}
	gomega.NewGomegaWithT(t).Eventually(func() bool {
		it, err := onRamp.FilterCCIPSendRequested(nil)
		require.NoError(t, err)
		for it.Next() {
			if it.Event.Message.SequenceNumber == seqNum {
				t.Log("sendRequested generated for", seqNum)
				return true
			}
		}
		c.Source.Chain.Commit()
		c.Dest.Chain.Commit()
		return false
	}, testutils.WaitTimeout(t), time.Second).Should(gomega.BeTrue(), "sendRequested has not been generated")
}

func (c *CCIPIntegrationTestHarness) ConsistentlyReportNotCommitted(t *testing.T, max int, commitStoreOpts ...common.Address) {
	var commitStore *commit_store.CommitStore
	var err error
	if len(commitStoreOpts) > 0 {
		commitStore, err = commit_store.NewCommitStore(commitStoreOpts[0], c.Dest.Chain.Client())
		require.NoError(t, err)
	} else {
		require.NotNil(t, c.Dest.CommitStore, "no commitStore configured")
		commitStore = c.Dest.CommitStore
	}
	gomega.NewGomegaWithT(t).Consistently(func() bool {
		minSeqNum, err := commitStore.GetExpectedNextSequenceNumber(nil)
		require.NoError(t, err)
		c.Source.Chain.Commit()
		c.Dest.Chain.Commit()
		t.Log("min seq num reported", minSeqNum)
		return minSeqNum > uint64(max)
	}, testutils.WaitTimeout(t), time.Second).Should(gomega.BeFalse(), "report has been committed")
}

func (c *CCIPIntegrationTestHarness) SetupAndStartNodes(ctx context.Context, t *testing.T, bootstrapNodePort int64) (Node, []Node, uint64) {
	appBootstrap, bootstrapPeerID, bootstrapTransmitter, bootstrapKb := setupNodeCCIP(t, c.Dest.User, bootstrapNodePort,
		"bootstrap_ccip", c.Source.Chain, c.Dest.Chain, big.NewInt(0).SetUint64(c.Source.ChainID),
		big.NewInt(0).SetUint64(c.Dest.ChainID), "", 0, c.Source.FinalityDepth,
		c.Dest.FinalityDepth)
	var (
		oracles []confighelper.OracleIdentityExtra
		nodes   []Node
	)
	err := appBootstrap.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, appBootstrap.Stop())
	})
	bootstrapNode := Node{
		App:         appBootstrap,
		Transmitter: bootstrapTransmitter,
		KeyBundle:   bootstrapKb,
	}
	// Set up the minimum 4 oracles all funded with destination ETH
	for i := int64(0); i < 4; i++ {
		app, peerID, transmitter, kb := setupNodeCCIP(
			t,
			c.Dest.User,
			int64(freeport.GetOne(t)),
			fmt.Sprintf("oracle_ccip%d", i),
			c.Source.Chain,
			c.Dest.Chain,
			big.NewInt(0).SetUint64(c.Source.ChainID),
			big.NewInt(0).SetUint64(c.Dest.ChainID),
			bootstrapPeerID,
			bootstrapNodePort,
			c.Source.FinalityDepth,
			c.Dest.FinalityDepth,
		)
		nodes = append(nodes, Node{
			App:         app,
			Transmitter: transmitter,
			KeyBundle:   kb,
		})
		offchainPublicKey, _ := hex.DecodeString(strings.TrimPrefix(kb.OnChainPublicKey(), "0x"))
		oracles = append(oracles, confighelper.OracleIdentityExtra{
			OracleIdentity: confighelper.OracleIdentity{
				OnchainPublicKey:  offchainPublicKey,
				TransmitAccount:   types4.Account(transmitter.String()),
				OffchainPublicKey: kb.OffchainPublicKey(),
				PeerID:            peerID,
			},
			ConfigEncryptionPublicKey: kb.ConfigEncryptionPublicKey(),
		})
		err = app.Start(ctx)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, app.Stop())
		})
	}

	c.Oracles = oracles
	commitOnchainConfig := c.CreateDefaultCommitOnchainConfig(t)
	commitOffchainConfig := c.CreateDefaultCommitOffchainConfig(t)
	execOnchainConfig := c.CreateDefaultExecOnchainConfig(t)
	execOffchainConfig := c.CreateDefaultExecOffchainConfig(t)

	configBlock := c.SetupOnchainConfig(t, commitOnchainConfig, commitOffchainConfig, execOnchainConfig, execOffchainConfig)
	c.Nodes = nodes
	c.Bootstrap = bootstrapNode
	//nolint:gosec // G115
	return bootstrapNode, nodes, uint64(configBlock)
}

// setup Jobs
func (c *CCIPIntegrationTestHarness) SetUpNodesAndJobs(t *testing.T, pricePipeline string, priceGetterConfig string, usdcAttestationAPI string) CCIPJobSpecParams {
	// Starts nodes and configures them in the OCR contracts.
	bootstrapNode, _, configBlock := c.SetupAndStartNodes(t.Context(), t, int64(freeport.GetOne(t)))

	jobParams := c.NewCCIPJobSpecParams(pricePipeline, priceGetterConfig, configBlock, usdcAttestationAPI)

	// Add the bootstrap job
	c.Bootstrap.AddBootstrapJob(t, jobParams.BootstrapJob(c.Dest.CommitStore.Address().Hex()))
	c.AddAllJobs(t, jobParams)

	// Replay for bootstrap.
	bs, err := bootstrapNode.App.GetRelayers().LegacyEVMChains().Get(strconv.FormatUint(c.Dest.ChainID, 10))
	require.NoError(t, err)
	require.LessOrEqual(t, configBlock, uint64(math.MaxInt64))
	bc, ok := bs.(legacyevm.Chain)
	require.True(t, ok)
	require.NoError(t, bc.LogPoller().Replay(t.Context(), int64(configBlock))) //nolint:gosec // G115 false positive
	c.Dest.Chain.Commit()

	return jobParams
}
func DecodeCommitOnChainConfig(encoded []byte) (ccipdata.CommitOnchainConfig, error) {
	var onchainConfig ccipdata.CommitOnchainConfig
	unpacked, err := abihelpers.DecodeOCR2Config(encoded)
	if err != nil {
		return onchainConfig, err
	}
	onChainCfg := unpacked.OnchainConfig
	onchainConfig, err = abihelpers.DecodeAbiStruct[ccipdata.CommitOnchainConfig](onChainCfg)
	if err != nil {
		return onchainConfig, err
	}
	return onchainConfig, nil
}

func DecodeExecOnChainConfig(encoded []byte) (v1_5_0.ExecOnchainConfig, error) {
	var onchainConfig v1_5_0.ExecOnchainConfig
	unpacked, err := abihelpers.DecodeOCR2Config(encoded)
	if err != nil {
		return onchainConfig, errors.Wrap(err, "failed to unpack log data")
	}
	onChainCfg := unpacked.OnchainConfig
	onchainConfig, err = abihelpers.DecodeAbiStruct[v1_5_0.ExecOnchainConfig](onChainCfg)
	if err != nil {
		return onchainConfig, err
	}
	return onchainConfig, nil
}

type ksa struct {
	keystore.Master
	csa keystore.CSA
}

func (k *ksa) CSA() keystore.CSA {
	return k.csa
}

func NewKsa(db *sqlx.DB, lggr logger.Logger, csa keystore.CSA) *ksa {
	return &ksa{
		Master: keystore.New(db, clutils.FastScryptParams, lggr),
		csa:    csa,
	}
}

type NoopFeedsClient struct{}

func (n NoopFeedsClient) ApprovedJob(context.Context, *pb.ApprovedJobRequest) (*pb.ApprovedJobResponse, error) {
	return &pb.ApprovedJobResponse{}, nil
}

func (n NoopFeedsClient) Healthcheck(context.Context, *pb.HealthcheckRequest) (*pb.HealthcheckResponse, error) {
	return &pb.HealthcheckResponse{}, nil
}

func (n NoopFeedsClient) UpdateNode(context.Context, *pb.UpdateNodeRequest) (*pb.UpdateNodeResponse, error) {
	return &pb.UpdateNodeResponse{}, nil
}

func (n NoopFeedsClient) RejectedJob(context.Context, *pb.RejectedJobRequest) (*pb.RejectedJobResponse, error) {
	return &pb.RejectedJobResponse{}, nil
}

func (n NoopFeedsClient) CancelledJob(context.Context, *pb.CancelledJobRequest) (*pb.CancelledJobResponse, error) {
	return &pb.CancelledJobResponse{}, nil
}
