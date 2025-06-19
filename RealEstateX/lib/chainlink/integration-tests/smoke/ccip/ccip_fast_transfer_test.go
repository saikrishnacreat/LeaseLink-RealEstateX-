package ccip

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	chainselectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_5_0/rmn_contract"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_5_1/token_pool"
	cldf_chain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	evmChain "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink/deployment/ccip/shared/bindings"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-evm/gethwrappers/shared/generated/burn_mint_erc677"
	"github.com/smartcontractkit/chainlink-evm/gethwrappers/shared/generated/link_token"
	"github.com/smartcontractkit/chainlink-evm/pkg/utils"
	"github.com/smartcontractkit/chainlink-testing-framework/lib/blockchain"
	"github.com/smartcontractkit/chainlink-testing-framework/lib/logging"
	"github.com/smartcontractkit/chainlink/deployment/ccip/changeset"
	"github.com/smartcontractkit/chainlink/deployment/ccip/changeset/testhelpers"
	v1_5testhelpers "github.com/smartcontractkit/chainlink/deployment/ccip/changeset/testhelpers/v1_5"
	"github.com/smartcontractkit/chainlink/deployment/ccip/changeset/v1_5"
	"github.com/smartcontractkit/chainlink/deployment/ccip/changeset/v1_5_1"
	"github.com/smartcontractkit/chainlink/deployment/ccip/shared"
	usd_stablecoin "github.com/smartcontractkit/chainlink/deployment/ccip/shared/bindings/usd_stablecoin"
	"github.com/smartcontractkit/chainlink/deployment/ccip/shared/stateview"
	"github.com/smartcontractkit/chainlink/deployment/ccip/shared/stateview/evm"
	commonchangeset "github.com/smartcontractkit/chainlink/deployment/common/changeset"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink/deployment/environment/devenv"
	testsetups "github.com/smartcontractkit/chainlink/integration-tests/testsetups/ccip"
)

var (
	feeTokenLink   = "LINK"
	feeTokenNative = "NATIVE"
)

type balanceToken interface {
	BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error)
}

type balanceAssertion func(t *testing.T, sourceToken balanceToken, destinationToken balanceToken, address common.Address, description string)

type fastTransferE2ETestCase struct {
	name                                string
	enableFiller                        bool
	allowlistEnabled                    bool
	allowlistFiller                     bool
	tokenSymbol                         string
	preFastTransferFillerAssertions     []balanceAssertion
	postFastTransferFillerAssertions    []balanceAssertion
	postRegularTransferFillerAssertions []balanceAssertion
	preFastTransferUserAssertions       []balanceAssertion
	postFastTransferUserAssertions      []balanceAssertion
	postRegularTransferUserAssertions   []balanceAssertion
	preFastTransferPoolAssertions       []balanceAssertion
	postFastTransferPoolAssertions      []balanceAssertion
	postRegularTransferPoolAssertions   []balanceAssertion
	feeTokenType                        string // "LINK" or "NATIVE"
	fastTransferPoolFeeBps              uint16
	externalMinter                      bool
}

var (
	initialFillerTokenAmountOnDest = big.NewInt(0).Mul(big.NewInt(1e18), big.NewInt(1000))
	initialUserTokenAmountOnSource = big.NewInt(200000)
	defaultEthAmount               = big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(1000))
	transferAmount                 = big.NewInt(100000)
	expectedFastTransferFee        = big.NewInt(100)
	tokenDecimals                  = uint8(18)
	sourceChainID                  = uint64(1337)
	destinationChainID             = uint64(2337)
)

type fastTransferE2ETestCaseOption func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase

func ftfTc(name string, options ...fastTransferE2ETestCaseOption) *fastTransferE2ETestCase {
	tc := &fastTransferE2ETestCase{
		name:         name,
		enableFiller: true,
		preFastTransferFillerAssertions: []balanceAssertion{
			assertDestinationBalanceEqual(initialFillerTokenAmountOnDest),
		},
		preFastTransferUserAssertions: []balanceAssertion{
			assertSourceBalanceEqual(initialUserTokenAmountOnSource),
		},
		postFastTransferFillerAssertions:    []balanceAssertion{},
		postRegularTransferFillerAssertions: []balanceAssertion{},
		postFastTransferUserAssertions:      []balanceAssertion{},
		postRegularTransferUserAssertions:   []balanceAssertion{},
		preFastTransferPoolAssertions:       []balanceAssertion{},
		postFastTransferPoolAssertions:      []balanceAssertion{},
		postRegularTransferPoolAssertions:   []balanceAssertion{},
		feeTokenType:                        feeTokenLink,
		fastTransferPoolFeeBps:              0,
	}

	for _, option := range options {
		tc = option(tc)
	}

	return tc
}

func withFillerDisabled() fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		tc.enableFiller = false
		return tc
	}
}

func withFastFillSuccessAmountAssertions() fastTransferE2ETestCaseOption {
	transferAmountMinusFee := big.NewInt(0).Sub(transferAmount, expectedFastTransferFee)
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		// Calculate pool fee: (transferAmount * fastTransferPoolFeeBps) / 10000
		poolFee := big.NewInt(0).Mul(transferAmount, big.NewInt(int64(tc.fastTransferPoolFeeBps)))
		poolFee = big.NewInt(0).Div(poolFee, big.NewInt(10000))
		userReceivedAmount := big.NewInt(0).Sub(transferAmountMinusFee, poolFee)

		// Filler assertions
		tc.postRegularTransferFillerAssertions = append(tc.postRegularTransferFillerAssertions, assertDestinationBalanceEventuallyEqual(big.NewInt(0).Add(initialFillerTokenAmountOnDest, expectedFastTransferFee)))
		tc.postFastTransferFillerAssertions = append(tc.postFastTransferFillerAssertions, assertDestinationBalanceEventuallyEqual(big.NewInt(0).Sub(initialFillerTokenAmountOnDest, userReceivedAmount)))

		// User assertions
		tc.postFastTransferUserAssertions = append(tc.postFastTransferUserAssertions, assertDestinationBalanceEventuallyEqual(userReceivedAmount))
		tc.postRegularTransferUserAssertions = append(tc.postRegularTransferUserAssertions, assertDestinationBalanceEventuallyEqual(userReceivedAmount))

		// Pool assertions
		tc.preFastTransferPoolAssertions = append(tc.preFastTransferPoolAssertions, assertDestinationBalanceEqual(big.NewInt(0)))
		tc.postFastTransferPoolAssertions = append(tc.postFastTransferPoolAssertions, assertDestinationBalanceEventuallyEqual(big.NewInt(0)))
		tc.postRegularTransferPoolAssertions = append(tc.postRegularTransferPoolAssertions, assertDestinationBalanceEqual(poolFee))

		return tc
	}
}

func withFastFillNoFillerSuccessAmountAssertions() fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		// Filler assertions
		tc.postFastTransferFillerAssertions = append(tc.postFastTransferFillerAssertions, assertDestinationBalanceEqual(initialFillerTokenAmountOnDest))
		tc.postRegularTransferFillerAssertions = append(tc.postRegularTransferFillerAssertions, assertDestinationBalanceEqual(initialFillerTokenAmountOnDest))

		// User assertions
		tc.postFastTransferUserAssertions = append(tc.postFastTransferUserAssertions, assertDestinationBalanceEventuallyEqual(big.NewInt(0)))
		tc.postRegularTransferUserAssertions = append(tc.postRegularTransferUserAssertions, assertDestinationBalanceEventuallyEqual(transferAmount))

		// Pool assertions
		tc.preFastTransferPoolAssertions = append(tc.preFastTransferPoolAssertions, assertDestinationBalanceEqual(big.NewInt(0)))
		tc.postFastTransferPoolAssertions = append(tc.postFastTransferPoolAssertions, assertDestinationBalanceEventuallyEqual(big.NewInt(0)))
		tc.postRegularTransferPoolAssertions = append(tc.postRegularTransferPoolAssertions, assertDestinationBalanceEqual(big.NewInt(0)))

		return tc
	}
}

func withFeeTokenType(feeTokenType string) fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		tc.feeTokenType = feeTokenType
		return tc
	}
}

func withFillerAllowlistEnabled() fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		tc.allowlistEnabled = true
		return tc
	}
}

func withAllowlistFiller() fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		tc.allowlistFiller = true
		return tc
	}
}

func withPoolFeeBps(poolFeeBps uint16) fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		tc.fastTransferPoolFeeBps = poolFeeBps
		return tc
	}
}

func withExternalMinter() fastTransferE2ETestCaseOption {
	return func(tc *fastTransferE2ETestCase) *fastTransferE2ETestCase {
		tc.externalMinter = true
		return tc
	}
}

var fastTransferTestCases = []*fastTransferE2ETestCase{
	ftfTc("fee token", withFeeTokenType(feeTokenLink), withFastFillSuccessAmountAssertions()),
	ftfTc("fee token and no filler", withFeeTokenType(feeTokenLink), withFastFillNoFillerSuccessAmountAssertions(), withFillerDisabled()),
	ftfTc("native fee token", withFeeTokenType(feeTokenNative), withFastFillSuccessAmountAssertions()),
	ftfTc("native fee token and no filler", withFeeTokenType(feeTokenNative), withFastFillNoFillerSuccessAmountAssertions(), withFillerDisabled()),
	ftfTc("allowlist enabled", withFillerAllowlistEnabled(), withAllowlistFiller(), withFastFillSuccessAmountAssertions()),
	ftfTc("allowlist enabled and filler not on allowlist", withFillerAllowlistEnabled(), withFastFillNoFillerSuccessAmountAssertions()),
	ftfTc("pool fee with filler", withPoolFeeBps(50), withFastFillSuccessAmountAssertions()),
	ftfTc("pool fee without filler", withPoolFeeBps(50), withFastFillNoFillerSuccessAmountAssertions(), withFillerDisabled()),
	ftfTc("external minter", withExternalMinter(), withFastFillSuccessAmountAssertions(), withFeeTokenType(feeTokenNative)),
	ftfTc("external minter feeToken", withExternalMinter(), withFastFillSuccessAmountAssertions(), withFeeTokenType(feeTokenLink)),
}

func assertDestinationBalanceEventuallyEqual(expectedBalance *big.Int) balanceAssertion {
	return func(t *testing.T, sourceToken balanceToken, destinationToken balanceToken, address common.Address, description string) {
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			balance, err := destinationToken.BalanceOf(nil, address)
			assert.NoError(collect, err)
			assert.Equal(collect, expectedBalance.Int64(), balance.Int64(), "Balance should be equal to expected value")
		}, 30*time.Second, time.Second, description+" - Balance should eventually be equal to expected value")
	}
}

func assertSourceBalanceEqual(expectedBalance *big.Int) balanceAssertion {
	return func(t *testing.T, sourceToken balanceToken, destinationToken balanceToken, address common.Address, descriptuon string) {
		balance, err := sourceToken.BalanceOf(nil, address)
		require.NoError(t, err)
		require.Equal(t, expectedBalance.Int64(), balance.Int64(), descriptuon+" - Balance should be equal to expected value")
	}
}

func assertDestinationBalanceEqual(expectedBalance *big.Int) balanceAssertion {
	return func(t *testing.T, sourceToken balanceToken, destinationToken balanceToken, address common.Address, description string) {
		balance, err := destinationToken.BalanceOf(nil, address)
		require.NoError(t, err)
		require.Equal(t, expectedBalance.Int64(), balance.Int64(), description+" - Balance should be equal to expected value")
	}
}

func createAccount(t *testing.T, chainID uint64) (common.Address, func() *bind.TransactOpts, *ecdsa.PrivateKey) {
	userPrivateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	userAddress := crypto.PubkeyToAddress(userPrivateKey.PublicKey)
	transactor := func() *bind.TransactOpts {
		userTransactor, err := bind.NewKeyedTransactorWithChainID(userPrivateKey, new(big.Int).SetUint64(chainID))
		require.NoError(t, err)
		return userTransactor
	}
	return userAddress, transactor, userPrivateKey
}

func deployTokenAndGrantAllRoles(t *testing.T, chain evmChain.Chain, tokenSymbol string, tokenDecimals uint8, lock *sync.Mutex, isExternalMinterToken bool) token {
	lock.Lock()
	defer lock.Unlock()

	if isExternalMinterToken {
		_, tx, token, err := usd_stablecoin.DeployStablecoin(
			chain.DeployerKey,
			chain.Client,
		)
		require.NoError(t, err)
		_, err = chain.Confirm(tx)
		require.NoError(t, err)

		tx, err = token.Initialize(chain.DeployerKey, tokenSymbol, tokenSymbol)
		require.NoError(t, err)
		_, err = chain.Confirm(tx)
		require.NoError(t, err)

		return token
	}

	_, tx, token, err := burn_mint_erc677.DeployBurnMintERC677(
		chain.DeployerKey,
		chain.Client,
		tokenSymbol,
		tokenSymbol,
		tokenDecimals,
		big.NewInt(0).Mul(big.NewInt(1e9), big.NewInt(1e18)),
	)
	require.NoError(t, err)
	_, err = chain.Confirm(tx)
	require.NoError(t, err)

	tx, err = token.GrantMintAndBurnRoles(chain.DeployerKey, chain.DeployerKey.From)
	require.NoError(t, err)
	_, err = chain.Confirm(tx)
	require.NoError(t, err)

	return token
}

func getLinkTokenAndGrantMintRole(t *testing.T, chain evmChain.Chain, state evm.CCIPChainState, sendLock *sync.Mutex) *link_token.LinkToken {
	sendLock.Lock()
	defer sendLock.Unlock()
	linkToken := state.LinkToken
	tx, err := linkToken.GrantMintRole(chain.DeployerKey, chain.DeployerKey.From)
	require.NoError(t, err)
	_, err = chain.Confirm(tx)
	require.NoError(t, err)

	return linkToken
}

type mintableToken interface {
	Mint(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error)
	Address() common.Address
}

type token interface {
	balanceToken
	approvableToken
	Address() common.Address
}

type sequenceNumberRetriever func(opts *bind.CallOpts, destChainSelector uint64) (uint64, error)
type waitForExecutionFn func(t *testing.T, sequenceNumber uint64)

func fundAccountWithToken(t *testing.T, chain evmChain.Chain, receiver common.Address, token mintableToken, amount *big.Int, sendLock *sync.Mutex) {
	sendLock.Lock()
	defer sendLock.Unlock()
	tx, err := token.Mint(chain.DeployerKey, receiver, amount)
	require.NoError(t, err)
	_, err = chain.Confirm(tx)
	require.NoError(t, err)
}

type approvableToken interface {
	Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error)
}

func approveToken(t *testing.T, chain evmChain.Chain, transactor *bind.TransactOpts, token approvableToken, spender common.Address) {
	tx, err := token.Approve(transactor, spender, big.NewInt(0).Mul(big.NewInt(1e18), big.NewInt(1e9))) // Approve a large amount
	require.NoError(t, err)
	_, err = chain.Confirm(tx)
	require.NoError(t, err)
}

type tokenPoolConfig struct {
	poolConfig        map[uint64]v1_5_1.DeployTokenPoolInput
	sourceMinter      mintableToken
	destinationMinter mintableToken
	postSetupAction   func(sourceTokenPool common.Address, destinationTokenPool common.Address)
	version           semver.Version
	poolType          cldf.ContractType
}

func configureExternalMinterTokenPool(t *testing.T, e cldf.Environment, sourceChainSelector, destinationChainSelector uint64, sourceTokenAddress, destinationTokenAddress common.Address, tokenDecimals uint8) tokenPoolConfig {
	sourceChain := e.BlockChains.EVMChains()[sourceChainSelector]
	destChain := e.BlockChains.EVMChains()[destinationChainSelector]

	_, sourceTokenGovernor := testhelpers.DeployTokenGovernor(t, e, sourceChainSelector, sourceTokenAddress)
	_, destinationTokenGovernor := testhelpers.DeployTokenGovernor(t, e, destinationChainSelector, destinationTokenAddress)

	bridgeBurnMintRole, err := sourceTokenGovernor.BRIDGEMINTERORBURNERROLE(nil)
	require.NoError(t, err)

	poolConfig := map[uint64]v1_5_1.DeployTokenPoolInput{
		sourceChainSelector: {
			Type:               shared.BurnMintWithExternalMinterFastTransferTokenPool,
			TokenAddress:       sourceTokenAddress,
			AllowList:          nil,
			LocalTokenDecimals: tokenDecimals,
			AcceptLiquidity:    nil,
			ExternalMinter:     sourceTokenGovernor.Address(),
		},
		destinationChainSelector: {
			Type:               shared.BurnMintWithExternalMinterFastTransferTokenPool,
			TokenAddress:       destinationTokenAddress,
			AllowList:          nil,
			LocalTokenDecimals: tokenDecimals,
			AcceptLiquidity:    nil,
			ExternalMinter:     destinationTokenGovernor.Address(),
		},
	}

	postSetupAction := func(sourceTokenPool common.Address, destinationTokenPool common.Address) {
		tx, err := sourceTokenGovernor.GrantRole(sourceChain.DeployerKey, bridgeBurnMintRole, sourceTokenPool)
		require.NoError(t, err)
		_, err = sourceChain.Confirm(tx)
		require.NoError(t, err)
		tx, err = destinationTokenGovernor.GrantRole(destChain.DeployerKey, bridgeBurnMintRole, destinationTokenPool)
		require.NoError(t, err)
		_, err = destChain.Confirm(tx)
		require.NoError(t, err)

		sourceToken, err := usd_stablecoin.NewStablecoin(sourceTokenAddress, sourceChain.Client)
		require.NoError(t, err)
		tx, err = sourceToken.TransferOwnership(sourceChain.DeployerKey, sourceTokenGovernor.Address())
		require.NoError(t, err)
		_, err = sourceChain.Confirm(tx)
		require.NoError(t, err)

		tx, err = sourceTokenGovernor.AcceptOwnership(sourceChain.DeployerKey)
		require.NoError(t, err)
		_, err = sourceChain.Confirm(tx)
		require.NoError(t, err)

		destinationToken, err := usd_stablecoin.NewStablecoin(destinationTokenAddress, destChain.Client)
		require.NoError(t, err)
		tx, err = destinationToken.TransferOwnership(destChain.DeployerKey, destinationTokenGovernor.Address())
		require.NoError(t, err)
		_, err = destChain.Confirm(tx)
		require.NoError(t, err)
		tx, err = destinationTokenGovernor.AcceptOwnership(destChain.DeployerKey)
		require.NoError(t, err)
		_, err = destChain.Confirm(tx)
		require.NoError(t, err)

		minterRole, err := sourceTokenGovernor.MINTERROLE(nil)
		require.NoError(t, err)
		tx, err = sourceTokenGovernor.GrantRole(sourceChain.DeployerKey, minterRole, sourceChain.DeployerKey.From)
		require.NoError(t, err)
		_, err = sourceChain.Confirm(tx)
		require.NoError(t, err)
		tx, err = destinationTokenGovernor.GrantRole(destChain.DeployerKey, minterRole, destChain.DeployerKey.From)
		require.NoError(t, err)
		_, err = destChain.Confirm(tx)
		require.NoError(t, err)
	}

	return tokenPoolConfig{
		poolConfig:        poolConfig,
		sourceMinter:      sourceTokenGovernor,
		destinationMinter: destinationTokenGovernor,
		postSetupAction:   postSetupAction,
		version:           shared.BurnMintWithExternalMinterFastTransferTokenPoolVersion,
		poolType:          shared.BurnMintWithExternalMinterFastTransferTokenPool,
	}
}

func configureBurnMintTokenPool(t *testing.T, e cldf.Environment, sourceChainSelector, destinationChainSelector uint64, sourceTokenAddress, destinationTokenAddress common.Address, tokenDecimals uint8) tokenPoolConfig {
	sourceChain := e.BlockChains.EVMChains()[sourceChainSelector]
	destChain := e.BlockChains.EVMChains()[destinationChainSelector]

	poolConfig := map[uint64]v1_5_1.DeployTokenPoolInput{
		sourceChainSelector: {
			Type:               shared.BurnMintFastTransferTokenPool,
			TokenAddress:       sourceTokenAddress,
			AllowList:          nil,
			LocalTokenDecimals: tokenDecimals,
			AcceptLiquidity:    nil,
		},
		destinationChainSelector: {
			Type:               shared.BurnMintFastTransferTokenPool,
			TokenAddress:       destinationTokenAddress,
			AllowList:          nil,
			LocalTokenDecimals: tokenDecimals,
			AcceptLiquidity:    nil,
		},
	}

	sourceToken, err := burn_mint_erc677.NewBurnMintERC677(sourceTokenAddress, sourceChain.Client)
	require.NoError(t, err)
	destToken, err := burn_mint_erc677.NewBurnMintERC677(destinationTokenAddress, destChain.Client)
	require.NoError(t, err)

	postSetupAction := func(sourceTokenPool common.Address, destinationTokenPool common.Address) {
		sourceTokenInstance, err := burn_mint_erc677.NewBurnMintERC677(sourceTokenAddress, sourceChain.Client)
		require.NoError(t, err)
		tx, err := sourceTokenInstance.GrantBurnRole(sourceChain.DeployerKey, sourceTokenPool)
		require.NoError(t, err)
		_, err = sourceChain.Confirm(tx)
		require.NoError(t, err)

		tx, err = destToken.GrantMintRole(destChain.DeployerKey, destinationTokenPool)
		require.NoError(t, err)
		_, err = destChain.Confirm(tx)
		require.NoError(t, err)
	}

	return tokenPoolConfig{
		poolConfig:        poolConfig,
		sourceMinter:      sourceToken,
		destinationMinter: destToken,
		postSetupAction:   postSetupAction,
		version:           shared.FastTransferTokenPoolVersion,
		poolType:          shared.BurnMintFastTransferTokenPool,
	}
}

func configureTokenPoolRateLimits(e cldf.Environment, tokenSymbol string, sourceChainSelector, destinationChainSelector uint64, poolType cldf.ContractType, version semver.Version) error {
	ratelimiterConfig := token_pool.RateLimiterConfig{
		IsEnabled: true,
		Capacity:  new(big.Int).Mul(big.NewInt(1e16), big.NewInt(2)),
		Rate:      big.NewInt(1),
	}
	tokenPoolConfig := map[uint64]v1_5_1.TokenPoolConfig{
		sourceChainSelector: {
			Type:    poolType,
			Version: version,
			ChainUpdates: v1_5_1.RateLimiterPerChain{
				destinationChainSelector: v1_5_1.RateLimiterConfig{
					Inbound:  ratelimiterConfig,
					Outbound: ratelimiterConfig,
				},
			},
		},
		destinationChainSelector: {
			Type:    poolType,
			Version: version,
			ChainUpdates: v1_5_1.RateLimiterPerChain{
				sourceChainSelector: v1_5_1.RateLimiterConfig{
					Inbound:  ratelimiterConfig,
					Outbound: ratelimiterConfig,
				},
			},
		},
	}
	_, err := v1_5_1.ConfigureTokenPoolContractsChangeset(e, v1_5_1.ConfigureTokenPoolContractsConfig{
		TokenSymbol: shared.TokenSymbol(tokenSymbol),
		PoolUpdates: tokenPoolConfig,
	})
	return err
}

func configureTokenAdminRegistry(e cldf.Environment, tokenSymbol string, sourceChainSelector, destinationChainSelector uint64, poolType cldf.ContractType, version semver.Version) error {
	registryConfig := map[uint64]map[shared.TokenSymbol]v1_5_1.TokenPoolInfo{
		sourceChainSelector: {
			shared.TokenSymbol(tokenSymbol): {
				Type:          poolType,
				Version:       version,
				ExternalAdmin: e.BlockChains.EVMChains()[sourceChainSelector].DeployerKey.From,
			},
		},
		destinationChainSelector: {
			shared.TokenSymbol(tokenSymbol): {
				Type:          poolType,
				Version:       version,
				ExternalAdmin: e.BlockChains.EVMChains()[destinationChainSelector].DeployerKey.From,
			},
		},
	}

	_, err := v1_5_1.ProposeAdminRoleChangeset(e, v1_5_1.TokenAdminRegistryChangesetConfig{
		Pools:                   registryConfig,
		SkipOwnershipValidation: true,
	})
	if err != nil {
		return err
	}

	_, err = v1_5_1.AcceptAdminRoleChangeset(e, v1_5_1.TokenAdminRegistryChangesetConfig{
		Pools:                   registryConfig,
		SkipOwnershipValidation: true,
	})
	if err != nil {
		return err
	}

	_, err = v1_5_1.SetPoolChangeset(e, v1_5_1.TokenAdminRegistryChangesetConfig{
		Pools:                   registryConfig,
		SkipOwnershipValidation: true,
	})
	return err
}

func getFirstAddressFromChain(t *testing.T, addressBook cldf.AddressBook, chainSelector uint64) common.Address {
	addresses, err := addressBook.AddressesForChain(chainSelector)
	require.NoError(t, err)

	for addr := range addresses {
		return common.HexToAddress(addr)
	}

	require.Failf(t, "No addresses found for chain", "ChainSelector: %d", chainSelector)
	return common.Address{}
}

func configureFastTransferSettings(t *testing.T, e cldf.Environment, tokenSymbol string, sourceChainSelector, destinationChainSelector uint64, fillerAddress common.Address, tc *fastTransferE2ETestCase, poolType cldf.ContractType, version semver.Version) error {
	fillers := []common.Address{}
	if tc.allowlistEnabled && tc.allowlistFiller {
		fillers = append(fillers, fillerAddress)
	}

	if tc.allowlistFiller {
		_, err := commonchangeset.Apply(t, e, commonchangeset.Configure(
			v1_5_1.FastTransferFillerAllowlistChangeset,
			v1_5_1.FastTransferFillerAllowlistConfig{
				TokenSymbol:     shared.TokenSymbol(tokenSymbol),
				ContractType:    poolType,
				ContractVersion: version,
				Updates: map[uint64]v1_5_1.FillerAllowlistConfig{
					sourceChainSelector: {
						AddFillers:    fillers,
						RemoveFillers: []common.Address{},
					},
					destinationChainSelector: {
						AddFillers:    fillers,
						RemoveFillers: []common.Address{},
					},
				},
			}))
		if err != nil {
			return err
		}
	}
	settlementGasOverhead := uint32(200000)
	_, err := commonchangeset.Apply(t, e, commonchangeset.Configure(
		v1_5_1.FastTransferUpdateLaneConfigChangeset,
		v1_5_1.FastTransferUpdateLaneConfigConfig{
			TokenSymbol:     shared.TokenSymbol(tokenSymbol),
			ContractType:    poolType,
			ContractVersion: version,
			Updates: map[uint64](map[uint64]v1_5_1.UpdateLaneConfig){
				sourceChainSelector: {
					destinationChainSelector: {
						FastTransferFillerFeeBps: 10,
						FastTransferPoolFeeBps:   tc.fastTransferPoolFeeBps,
						FillerAllowlistEnabled:   tc.allowlistEnabled,
						FillAmountMaxRequest:     big.NewInt(100000),
						SettlementOverheadGas:    &settlementGasOverhead,
						SkipAllowlistValidation:  true,
					},
				},
				destinationChainSelector: {
					sourceChainSelector: {
						FastTransferFillerFeeBps: 20,
						FastTransferPoolFeeBps:   tc.fastTransferPoolFeeBps,
						FillerAllowlistEnabled:   tc.allowlistEnabled,
						FillAmountMaxRequest:     big.NewInt(100000),
						SettlementOverheadGas:    &settlementGasOverhead,
						SkipAllowlistValidation:  true,
					},
				},
			},
		}))
	return err
}

func configureTokenPoolContracts(t *testing.T, e cldf.Environment, tokenSymbol string, sourceChainSelector, destinationChainSelector uint64, sourceTokenAddress, destinationTokenAddress common.Address, tokenDecimals uint8, fillerAddress common.Address, tc *fastTransferE2ETestCase, sourceLock *sync.Mutex, destinationLock *sync.Mutex) (sourcePoolAddr common.Address, destPoolAddr common.Address, version semver.Version, poolWrapper *bindings.FastTransferTokenPoolWrapper, sourceMinter mintableToken, destMinter mintableToken) {
	sourceLock.Lock()
	defer sourceLock.Unlock()
	destinationLock.Lock()
	defer destinationLock.Unlock()

	var config tokenPoolConfig
	if tc.externalMinter {
		config = configureExternalMinterTokenPool(t, e, sourceChainSelector, destinationChainSelector, sourceTokenAddress, destinationTokenAddress, tokenDecimals)
	} else {
		config = configureBurnMintTokenPool(t, e, sourceChainSelector, destinationChainSelector, sourceTokenAddress, destinationTokenAddress, tokenDecimals)
	}

	cs, err := v1_5_1.DeployTokenPoolContractsChangeset(e, v1_5_1.DeployTokenPoolContractsConfig{
		TokenSymbol: shared.TokenSymbol(tokenSymbol),
		NewPools:    config.poolConfig,
	})
	require.NoError(t, err)

	sourceTokenPoolAddress := getFirstAddressFromChain(t, cs.AddressBook, sourceChainSelector)           //nolint:staticcheck // AddressBook is deprecated but still required
	destinationTokenPoolAddress := getFirstAddressFromChain(t, cs.AddressBook, destinationChainSelector) //nolint:staticcheck // AddressBook is deprecated but still required

	err = e.ExistingAddresses.Merge(cs.AddressBook) //nolint:staticcheck // AddressBook is deprecated but still required
	require.NoError(t, err)

	err = configureTokenPoolRateLimits(e, tokenSymbol, sourceChainSelector, destinationChainSelector, config.poolType, config.version)
	require.NoError(t, err)

	err = configureTokenAdminRegistry(e, tokenSymbol, sourceChainSelector, destinationChainSelector, config.poolType, config.version)
	require.NoError(t, err)

	err = configureFastTransferSettings(t, e, tokenSymbol, sourceChainSelector, destinationChainSelector, fillerAddress, tc, config.poolType, config.version)
	require.NoError(t, err)

	sourceTokenPool, err := bindings.GetFastTransferTokenPoolContract(e, shared.TokenSymbol(tokenSymbol), config.poolType, config.version, sourceChainSelector)
	require.NoError(t, err)

	config.postSetupAction(sourceTokenPoolAddress, destinationTokenPoolAddress)

	sourcePoolAddr = sourceTokenPoolAddress
	destPoolAddr = destinationTokenPoolAddress
	version = config.version
	poolWrapper = sourceTokenPool
	sourceMinter = config.sourceMinter
	destMinter = config.destinationMinter
	return
}

func runAssertions(t *testing.T, sourceToken balanceToken, destinationToken balanceToken, address common.Address, assertions []balanceAssertion, description string) {
	for _, assertion := range assertions {
		assertion(t, sourceToken, destinationToken, address, description)
	}
}

func startRelayer(t *testing.T, sourceChainSelector, destinationChainSelector uint64, sourceTokenPoolAddress common.Address, destinationTokenPoolAddress common.Address, deployedEnv testhelpers.TestEnvironment, fillerPrivateKey *ecdsa.PrivateKey) func() error {
	dockerEnv, ok := deployedEnv.(*testsetups.DeployedLocalDevEnvironment)
	require.True(t, ok, "deployedEnv is not of type *testsetups.DeployedLocalDevEnvironment")

	networks := dockerEnv.GetCLClusterTestEnv().EVMNetworks
	var sourceChainNetwork *blockchain.EVMNetwork
	for _, network := range networks {
		if network.ChainID >= 0 && uint64(network.ChainID) == sourceChainID {
			sourceChainNetwork = network
			break
		}
	}
	require.NotNil(t, sourceChainNetwork, "Source chain network not found in EVM networks")

	var destinationChainNetwork *blockchain.EVMNetwork
	for _, network := range networks {
		if network.ChainID >= 0 && uint64(network.ChainID) == destinationChainID {
			destinationChainNetwork = network
			break
		}
	}
	require.NotNil(t, destinationChainNetwork, "Destination chain network not found in EVM networks")

	marshalledKey := crypto.FromECDSA(fillerPrivateKey)

	hexString := hex.EncodeToString(marshalledKey)

	fastFillerConfig := devenv.CCIPFastFillerConfig{
		SignerProviders: []devenv.SignerProvider{
			{
				Name:       "filler",
				Type:       "raw",
				PrivateKey: hexString,
			},
		},
		Listeners: []devenv.ListenerConfig{
			{
				ChainSelector:    strconv.FormatUint(sourceChainSelector, 10),
				TokenPoolAddress: sourceTokenPoolAddress.Hex(),
				RPCURL:           sourceChainNetwork.HTTPURLs[0],
			},
		},
		Fillers: []devenv.FillerConfig{
			{
				ChainSelector:    strconv.FormatUint(destinationChainSelector, 10),
				TokenPoolAddress: destinationTokenPoolAddress.Hex(),
				RPCURL:           destinationChainNetwork.HTTPURLs[0],
				SignerProvider:   "filler",
			},
		},
	}
	l := logging.GetTestLogger(t)
	relayer := devenv.NewCCIPFastFiller(fastFillerConfig, l, []string{dockerEnv.GetCLClusterTestEnv().DockerNetwork.ID})
	err := relayer.Start(t.Context(), t)
	require.NoError(t, err, "Failed to start the relayer")

	return func() error { return relayer.Stop(context.Background()) }
}

func TestFastTransfer1_5Lanes(t *testing.T) {
	e, _, tEnv := testsetups.NewIntegrationEnvironment(
		t,
		testhelpers.WithPrerequisiteDeploymentOnly(
			&changeset.V1_5DeploymentConfig{
				PriceRegStalenessThreshold: 60 * 60 * 24 * 14, // two weeks
				RMNConfig: &rmn_contract.RMNConfig{
					BlessWeightThreshold: 2,
					CurseWeightThreshold: 2,
					// setting dummy voters, we will permabless this later
					Voters: []rmn_contract.RMNVoter{
						{
							BlessWeight:   2,
							CurseWeight:   2,
							BlessVoteAddr: utils.RandomAddress(),
							CurseVoteAddr: utils.RandomAddress(),
						},
					},
				},
			}),
	)
	state, err := stateview.LoadOnchainState(e.Env)
	require.NoError(t, err)
	allChains := e.Env.BlockChains.ListChainSelectors(cldf_chain.WithFamily(chainselectors.FamilyEVM))
	src1, dest := allChains[0], allChains[1]
	pairs := []testhelpers.SourceDestPair{
		{SourceChainSelector: src1, DestChainSelector: dest},
		{SourceChainSelector: dest, DestChainSelector: src1},
	}
	// wire up all lanes
	// deploy onRamp, commit store, offramp , set ocr2config and send corresponding jobs
	e.Env = v1_5testhelpers.AddLanes(t, e.Env, state, pairs)

	// permabless the commit stores
	e.Env, err = commonchangeset.Apply(t, e.Env,
		commonchangeset.Configure(
			cldf.CreateLegacyChangeSet(v1_5.PermaBlessCommitStoreChangeset),
			v1_5.PermaBlessCommitStoreConfig{
				Configs: map[uint64]v1_5.PermaBlessCommitStoreConfigPerDest{
					dest: {
						Sources: []v1_5.PermaBlessConfigPerSourceChain{
							{
								SourceChainSelector: src1,
								PermaBless:          true,
							},
						},
					},
					src1: {
						Sources: []v1_5.PermaBlessConfigPerSourceChain{
							{
								SourceChainSelector: dest,
								PermaBless:          true,
							},
						},
					},
				},
			},
		),
	)
	require.NoError(t, err)

	onChainState, err := stateview.LoadOnchainState(e.Env)
	require.NoError(t, err)

	sourceChainSelector := e.Env.BlockChains.ListChainSelectors()[0]
	destinationChainSelector := e.Env.BlockChains.ListChainSelectors()[1]

	sourceChainState := onChainState.Chains[sourceChainSelector]
	destinationChain := e.Env.BlockChains.EVMChains()[destinationChainSelector]

	require.NoError(t, err)

	seqNumRetriever := func(opts *bind.CallOpts, destChainSelector uint64) (uint64, error) {
		onramp := onChainState.Chains[sourceChainSelector].EVM2EVMOnRamp[destChainSelector]
		seq, err := onramp.GetExpectedNextSequenceNumber(opts)
		if err != nil {
			return 0, fmt.Errorf("failed to get expected next sequence number: %w", err)
		}
		return seq, nil
	}

	offramp := onChainState.Chains[destinationChainSelector].EVM2EVMOffRamp[sourceChainSelector]
	commitStore := onChainState.Chains[destinationChainSelector].CommitStore[sourceChainSelector]

	waitForExecution := func(t *testing.T, sequenceNumber uint64) {
		sourceChain := e.Env.BlockChains.EVMChains()[sourceChainSelector]
		v1_5testhelpers.WaitForCommit(t, sourceChain, destinationChain, commitStore, sequenceNumber)
		e.Env.Logger.Infof("Commit confirmed, waiting for offramp execution for sequence number %d", sequenceNumber)
		v1_5testhelpers.WaitForExecute(t, sourceChain, destinationChain, offramp, []uint64{sequenceNumber}, uint64(0))
	}

	// Create shared locks for coordination between parallel test cases
	sourceLock := &sync.Mutex{}
	destinationLock := &sync.Mutex{}
	sendLock := &sync.Mutex{}

	for i, tc := range fastTransferTestCases {
		ctx := newFastTransferTestContext(
			e.Env,
			i,
			sourceChainState,
			tEnv,
			seqNumRetriever,
			waitForExecution,
			sourceLock,
			destinationLock,
			sendLock,
		)
		runFastTransferTestCase(t, ctx, tc)
	}
}

func TestFastTransfer1_6Lanes(t *testing.T) {
	e, _, deployedEnv := testsetups.NewIntegrationEnvironment(t)

	onChainState, err := stateview.LoadOnchainState(e.Env)
	require.NoError(t, err)
	testhelpers.AddLanesForAll(t, &e, onChainState)

	sourceChainSelector := e.Env.BlockChains.ListChainSelectors()[0]
	destinationChainSelector := e.Env.BlockChains.ListChainSelectors()[1]

	sourceChainState := onChainState.Chains[sourceChainSelector]
	destinationChain := e.Env.BlockChains.EVMChains()[destinationChainSelector]

	seqNumRetriever := func(opts *bind.CallOpts, destChainSelector uint64) (uint64, error) {
		onramp := onChainState.Chains[sourceChainSelector].OnRamp
		seq, err := onramp.GetExpectedNextSequenceNumber(opts, destChainSelector)
		if err != nil {
			return 0, fmt.Errorf("failed to get expected next sequence number: %w", err)
		}
		return seq, nil
	}

	offramp := onChainState.Chains[destinationChainSelector].OffRamp
	waitForExecution := func(t *testing.T, sequenceNumber uint64) {
		zero := uint64(0)
		_, _ = testhelpers.ConfirmExecWithSeqNrs(t, sourceChainSelector, destinationChain, offramp, &zero, []uint64{sequenceNumber})
	}

	// Create shared locks for coordination between parallel test cases
	sourceLock := &sync.Mutex{}
	destinationLock := &sync.Mutex{}
	sendLock := &sync.Mutex{}

	for i, tc := range fastTransferTestCases {
		ctx := newFastTransferTestContext(
			e.Env,
			i,
			sourceChainState,
			deployedEnv,
			seqNumRetriever,
			waitForExecution,
			sourceLock,
			destinationLock,
			sendLock,
		)
		runFastTransferTestCase(t, ctx, tc)
	}
}

type fastTransferTestContext struct {
	env                     cldf.Environment
	testIndex               int
	sourceLock              *sync.Mutex
	destinationLock         *sync.Mutex
	sendLock                *sync.Mutex
	sourceChainState        evm.CCIPChainState
	deployedEnv             testhelpers.TestEnvironment
	sequenceNumberRetriever sequenceNumberRetriever
	waitForExecution        waitForExecutionFn
}

func (ctx *fastTransferTestContext) SourceChainSelector() uint64 {
	return ctx.env.BlockChains.ListChainSelectors()[0]
}

func (ctx *fastTransferTestContext) DestinationChainSelector() uint64 {
	return ctx.env.BlockChains.ListChainSelectors()[1]
}

func (ctx *fastTransferTestContext) SourceChain() evmChain.Chain {
	return ctx.env.BlockChains.EVMChains()[ctx.SourceChainSelector()]
}

func (ctx *fastTransferTestContext) DestinationChain() evmChain.Chain {
	return ctx.env.BlockChains.EVMChains()[ctx.DestinationChainSelector()]
}

func (ctx *fastTransferTestContext) SourceLock() *sync.Mutex {
	return ctx.sourceLock
}

func (ctx *fastTransferTestContext) DestinationLock() *sync.Mutex {
	return ctx.destinationLock
}

func (ctx *fastTransferTestContext) SendLock() *sync.Mutex {
	return ctx.sendLock
}

func newFastTransferTestContext(
	env cldf.Environment,
	testIndex int,
	sourceChainState evm.CCIPChainState,
	deployedEnv testhelpers.TestEnvironment,
	sequenceNumberRetriever sequenceNumberRetriever,
	waitForExecution waitForExecutionFn,
	sourceLock *sync.Mutex,
	destinationLock *sync.Mutex,
	sendLock *sync.Mutex,
) *fastTransferTestContext {
	return &fastTransferTestContext{
		env:                     env,
		testIndex:               testIndex,
		sourceLock:              sourceLock,
		destinationLock:         destinationLock,
		sendLock:                sendLock,
		sourceChainState:        sourceChainState,
		deployedEnv:             deployedEnv,
		sequenceNumberRetriever: sequenceNumberRetriever,
		waitForExecution:        waitForExecution,
	}
}

func runFastTransferTestCase(t *testing.T, ctx *fastTransferTestContext, tc *fastTransferE2ETestCase) {
	tc.tokenSymbol = fmt.Sprintf("FTF_TEST_%d", ctx.testIndex+1)
	t.Run(tc.name, func(t *testing.T) {
		t.Parallel()
		userAddress, userTransactor, _ := createAccount(t, sourceChainID)
		fillerAddress, fillerTransactor, fillerPrivateKey := createAccount(t, destinationChainID)
		sourceToken := deployTokenAndGrantAllRoles(t, ctx.SourceChain(), tc.tokenSymbol, tokenDecimals, ctx.sourceLock, tc.externalMinter)
		destinationToken := deployTokenAndGrantAllRoles(t, ctx.DestinationChain(), tc.tokenSymbol, tokenDecimals, ctx.destinationLock, tc.externalMinter)

		sourceTokenPoolAddress, destinationTokenPoolAddress, _, _, sourceMinter, destinationMinter := configureTokenPoolContracts(t, ctx.env, tc.tokenSymbol, ctx.SourceChainSelector(), ctx.DestinationChainSelector(), sourceToken.Address(), destinationToken.Address(), tokenDecimals, fillerAddress, tc, ctx.sourceLock, ctx.destinationLock)
		var contractType cldf.ContractType
		if tc.externalMinter {
			contractType = shared.BurnMintWithExternalMinterFastTransferTokenPool
		} else {
			contractType = shared.BurnMintFastTransferTokenPool
		}
		pool, err := bindings.NewFastTransferTokenPoolWrapper(sourceTokenPoolAddress, ctx.SourceChain().Client, contractType)
		require.NoError(t, err)

		onChainState, err := stateview.LoadOnchainState(ctx.env)
		require.NoError(t, err)

		userEncodedAddress := common.LeftPadBytes(userAddress.Bytes(), 32)

		var feeTokenAddress common.Address
		switch tc.feeTokenType {
		case feeTokenLink:
			feeTokenAddress = onChainState.Chains[ctx.SourceChainSelector()].LinkToken.Address()
		case feeTokenNative:
			feeTokenAddress = common.HexToAddress("0x0")
		default:
			t.Fatalf("Unknown fee token type: %s", tc.feeTokenType)
		}

		fees, err := pool.GetCcipSendTokenFee(nil, ctx.DestinationChainSelector(), transferAmount, userEncodedAddress, feeTokenAddress, []byte{})
		require.NoError(t, err)

		// Setup source chain funding and approvals
		fundAccount(t, ctx.SourceChain(), userAddress, defaultEthAmount, ctx.sourceLock)
		fundAccountWithToken(t, ctx.SourceChain(), userAddress, sourceMinter, initialUserTokenAmountOnSource, ctx.sourceLock)
		approveToken(t, ctx.SourceChain(), userTransactor(), sourceToken, sourceTokenPoolAddress)

		if tc.feeTokenType == feeTokenLink {
			sourceLinkToken := getLinkTokenAndGrantMintRole(t, ctx.SourceChain(), ctx.sourceChainState, ctx.sourceLock)
			fundAccountWithToken(t, ctx.SourceChain(), userAddress, sourceLinkToken, fees.CcipSettlementFee, ctx.sourceLock)
			approveToken(t, ctx.SourceChain(), userTransactor(), sourceLinkToken, sourceTokenPoolAddress)
		}

		// Setup destination chain funding and approvals
		fundAccount(t, ctx.DestinationChain(), fillerAddress, defaultEthAmount, ctx.destinationLock)
		fundAccountWithToken(t, ctx.DestinationChain(), fillerAddress, destinationMinter, initialFillerTokenAmountOnDest, ctx.destinationLock)
		approveToken(t, ctx.DestinationChain(), fillerTransactor(), destinationToken, destinationTokenPoolAddress)

		if tc.enableFiller {
			stop := startRelayer(t, ctx.SourceChainSelector(), ctx.DestinationChainSelector(), sourceTokenPoolAddress, destinationTokenPoolAddress, ctx.deployedEnv, fillerPrivateKey)
			ctx.env.Logger.Infof("Started relayer for source chain %d and destination chain %d", ctx.SourceChainSelector(), ctx.DestinationChainSelector())

			defer func() {
				ctx.env.Logger.Infof("Stopping relayer for source chain %d and destination chain %d", ctx.SourceChainSelector(), ctx.DestinationChainSelector())
				_ = stop()
			}()
		}

		runAssertions(t, sourceToken, destinationToken, fillerAddress, tc.preFastTransferFillerAssertions, "Pre Fast Transfer Filler Assertions")
		runAssertions(t, sourceToken, destinationToken, userAddress, tc.preFastTransferUserAssertions, "Pre Fast Transfer User Assertions")
		runAssertions(t, sourceToken, destinationToken, destinationTokenPoolAddress, tc.preFastTransferPoolAssertions, "Pre Fast Transfer Pool Assertions")

		userTransac := userTransactor()
		if tc.feeTokenType == feeTokenNative {
			userTransac.Value = fees.CcipSettlementFee
		}

		var seqNum uint64
		func() {
			ctx.sendLock.Lock()
			defer ctx.sendLock.Unlock()
			require.NoError(t, err)
			seqNum, err = ctx.sequenceNumberRetriever(nil, ctx.DestinationChainSelector())
			require.NoError(t, err)
			ctx.env.Logger.Infof("Sending transaction from user address: %s", userTransac.From.Hex())
			tx, err := pool.CcipSendToken(userTransac, ctx.DestinationChainSelector(), transferAmount, fees.FastTransferFee, userEncodedAddress, feeTokenAddress, []byte{})
			ctx.env.Logger.Infof("Sending transaction: %s", tx.Hash().Hex())
			require.NoError(t, err)
			_, err = ctx.SourceChain().Confirm(tx)
			require.NoError(t, err)

			filter, err := pool.FilterFastTransferRequested(nil, nil, nil, nil)
			require.NoError(t, err)
			for filter.Next() {
				event := filter.Event()
				ctx.env.Logger.Infof("FastTransferRequested event: %s, fillId: %s, settlementId: %s", event.Raw.TxHash.Hex(), hex.EncodeToString(event.FillID[:]), hex.EncodeToString(event.SettlementID[:]))
			}
		}()

		runAssertions(t, sourceToken, destinationToken, fillerAddress, tc.postFastTransferFillerAssertions, "Post Fast Transfer Filler Assertions")
		runAssertions(t, sourceToken, destinationToken, userAddress, tc.postFastTransferUserAssertions, "Post Fast Transfer User Assertions")
		runAssertions(t, sourceToken, destinationToken, destinationTokenPoolAddress, tc.postFastTransferPoolAssertions, "Post Fast Transfer Pool Assertions")

		ctx.waitForExecution(t, seqNum)

		runAssertions(t, sourceToken, destinationToken, fillerAddress, tc.postRegularTransferFillerAssertions, "Post Regular Transfer Filler Assertions")
		runAssertions(t, sourceToken, destinationToken, userAddress, tc.postRegularTransferUserAssertions, "Post Regular Transfer User Assertions")
		runAssertions(t, sourceToken, destinationToken, destinationTokenPoolAddress, tc.postRegularTransferPoolAssertions, "Post Regular Transfer Pool Assertions")
	})
}

func fundAccount(
	t *testing.T,
	chain evmChain.Chain,
	receiver common.Address,
	amount *big.Int,
	sendLock *sync.Mutex,
) {
	sendLock.Lock()
	defer sendLock.Unlock()
	client := chain.Client
	sender := chain.DeployerKey

	nonce, err := client.NonceAt(t.Context(), sender.From, nil)
	require.NoError(t, err)

	gasPrice, err := client.SuggestGasPrice(t.Context())
	require.NoError(t, err)
	gasLimit := uint64(21000)

	tx := types.NewTransaction(nonce, receiver, amount, gasLimit, gasPrice, nil)

	signedTx, err := sender.Signer(sender.From, tx)
	require.NoError(t, err)

	err = client.SendTransaction(t.Context(), signedTx)
	require.NoError(t, err)

	_, err = chain.Confirm(signedTx)
	require.NoError(t, err)
}
