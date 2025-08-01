package pipeline

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/hex"
	clnull "github.com/smartcontractkit/chainlink-common/pkg/utils/null"
	"github.com/smartcontractkit/chainlink-evm/pkg/chains/legacyevm"
	"github.com/smartcontractkit/chainlink-evm/pkg/txmgr"
	txmgrcommon "github.com/smartcontractkit/chainlink-framework/chains/txmgr"
)

// Return types:
//
//	nil
type ETHTxTask struct {
	BaseTask         `mapstructure:",squash"`
	From             string `json:"from"`
	To               string `json:"to"`
	Data             string `json:"data"`
	GasLimit         string `json:"gasLimit"`
	TxMeta           string `json:"txMeta"`
	MinConfirmations string `json:"minConfirmations"`
	// FailOnRevert, if set, will error the task if the transaction reverted on-chain
	// If unset, the receipt will be passed as output
	// It has no effect if minConfirmations == 0
	FailOnRevert    string `json:"failOnRevert"`
	EVMChainID      string `json:"evmChainID" mapstructure:"evmChainID"`
	TransmitChecker string `json:"transmitChecker"`

	forwardingAllowed bool
	specGasLimit      *uint32
	keyStore          ETHKeyStore
	legacyChains      legacyevm.LegacyChainContainer
	jobType           string
}

type ETHKeyStore interface {
	GetRoundRobinAddress(ctx context.Context, chainID *big.Int, addrs ...common.Address) (common.Address, error)
}

var _ Task = (*ETHTxTask)(nil)

func (t *ETHTxTask) Type() TaskType {
	return TaskTypeETHTx
}

func (t *ETHTxTask) getEvmChainID() string {
	if t.EVMChainID == "" {
		t.EVMChainID = "$(jobSpec.evmChainID)"
	}
	return t.EVMChainID
}

func (t *ETHTxTask) Run(ctx context.Context, lggr logger.Logger, vars Vars, inputs []Result) (Result, RunInfo) {
	var chainID StringParam
	err := errors.Wrap(ResolveParam(&chainID, From(VarExpr(t.getEvmChainID(), vars), NonemptyString(t.getEvmChainID()), "")), "evmChainID")
	if err != nil {
		return Result{Error: err}, RunInfo{}
	}

	chain, err := t.legacyChains.Get(string(chainID))
	if err != nil {
		err = fmt.Errorf("%w: %s: %w", ErrInvalidEVMChainID, chainID, err)
		return Result{Error: err}, retryableRunInfo()
	}
	legacyChain, ok := chain.(legacyevm.Chain)
	if !ok {
		return Result{Error: ErrUnsupportedInLOOPPMode}, RunInfo{}
	}

	cfg := legacyChain.Config().EVM()
	txManager := legacyChain.TxManager()
	_, err = CheckInputs(inputs, -1, -1, 0)
	if err != nil {
		return Result{Error: errors.Wrap(err, "task inputs")}, RunInfo{}
	}

	maximumGasLimit := SelectGasLimit(cfg.GasEstimator(), t.jobType, t.specGasLimit)

	var (
		fromAddrs             AddressSliceParam
		toAddr                AddressParam
		data                  BytesParam
		gasLimit              Uint64Param
		txMetaMap             MapParam
		maybeMinConfirmations MaybeUint64Param
		transmitCheckerMap    MapParam
		failOnRevert          BoolParam
	)
	err = multierr.Combine(
		errors.Wrap(ResolveParam(&fromAddrs, From(VarExpr(t.From, vars), JSONWithVarExprs(t.From, vars, false), NonemptyString(t.From), nil)), "from"),
		errors.Wrap(ResolveParam(&toAddr, From(VarExpr(t.To, vars), NonemptyString(t.To))), "to"),
		errors.Wrap(ResolveParam(&data, From(VarExpr(t.Data, vars), NonemptyString(t.Data))), "data"),
		errors.Wrap(ResolveParam(&gasLimit, From(VarExpr(t.GasLimit, vars), NonemptyString(t.GasLimit), maximumGasLimit)), "gasLimit"),
		errors.Wrap(ResolveParam(&txMetaMap, From(VarExpr(t.TxMeta, vars), JSONWithVarExprs(t.TxMeta, vars, false), MapParam{})), "txMeta"),
		errors.Wrap(ResolveParam(&maybeMinConfirmations, From(VarExpr(t.MinConfirmations, vars), NonemptyString(t.MinConfirmations), "")), "minConfirmations"),
		errors.Wrap(ResolveParam(&transmitCheckerMap, From(VarExpr(t.TransmitChecker, vars), JSONWithVarExprs(t.TransmitChecker, vars, false), MapParam{})), "transmitChecker"),
		errors.Wrap(ResolveParam(&failOnRevert, From(NonemptyString(t.FailOnRevert), false)), "failOnRevert"),
	)
	if err != nil {
		return Result{Error: err}, RunInfo{}
	}
	minOutgoingConfirmations, isMinConfirmationSet := maybeMinConfirmations.Uint64()

	txMeta, err := decodeMeta(txMetaMap)
	if err != nil {
		return Result{Error: err}, RunInfo{}
	}
	txMeta.FailOnRevert = null.BoolFrom(bool(failOnRevert))
	setJobIDOnMeta(lggr, vars, txMeta)

	transmitChecker, err := decodeTransmitChecker(transmitCheckerMap)
	if err != nil {
		return Result{Error: err}, RunInfo{}
	}

	fromAddr, err := t.keyStore.GetRoundRobinAddress(ctx, legacyChain.ID(), fromAddrs...)
	if err != nil {
		err = errors.Wrap(err, "ETHTxTask failed to get fromAddress")
		lggr.Error(err)
		return Result{Error: errors.Wrapf(ErrTaskRunFailed, "while querying keystore: %v", err)}, retryableRunInfo()
	}

	// TODO(sc-55115): Allow job specs to pass in the strategy that they want
	strategy := txmgrcommon.NewSendEveryStrategy()

	var forwarderAddress common.Address
	if t.forwardingAllowed {
		var fwderr error
		forwarderAddress, fwderr = legacyChain.TxManager().GetForwarderForEOA(ctx, fromAddr)
		if fwderr != nil {
			lggr.Warnw("Skipping forwarding for job, will fallback to default behavior", "err", fwderr)
		}
	}

	txRequest := txmgr.TxRequest{
		FromAddress:      fromAddr,
		ToAddress:        common.Address(toAddr),
		EncodedPayload:   []byte(data),
		FeeLimit:         uint64(gasLimit),
		Meta:             txMeta,
		ForwarderAddress: forwarderAddress,
		Strategy:         strategy,
		Checker:          transmitChecker,
		SignalCallback:   true,
	}

	if !isMinConfirmationSet {
		// Store the task run ID, so we can resume the pipeline when tx is finalized
		txRequest.PipelineTaskRunID = &t.uuid
	} else if minOutgoingConfirmations > 0 {
		// Store the task run ID, so we can resume the pipeline after minOutgoingConfirmations
		txRequest.PipelineTaskRunID = &t.uuid
		txRequest.MinConfirmations = clnull.Uint32From(uint32(minOutgoingConfirmations))
	}

	_, err = txManager.CreateTransaction(ctx, txRequest)
	if err != nil {
		return Result{Error: errors.Wrapf(ErrTaskRunFailed, "while creating transaction: %v", err)}, retryableRunInfo()
	}

	if txRequest.PipelineTaskRunID != nil {
		return Result{}, RunInfo{IsPending: true}
	}

	return Result{}, RunInfo{}
}

func decodeMeta(metaMap MapParam) (*txmgr.TxMeta, error) {
	var txMeta txmgr.TxMeta
	metaDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &txMeta,
		ErrorUnused: true,
		DecodeHook: func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
			switch from {
			case stringType:
				switch to {
				case int32Type:
					i, err2 := strconv.ParseInt(data.(string), 10, 32)
					return int32(i), err2
				case reflect.TypeOf(common.Hash{}):
					hb, err := hex.DecodeString(data.(string))
					if err != nil {
						return nil, err
					}
					return common.BytesToHash(hb), nil
				}
			}
			return data, nil
		},
	})
	if err != nil {
		return &txMeta, errors.Wrapf(ErrBadInput, "txMeta: %v", err)
	}

	err = metaDecoder.Decode(metaMap)
	if err != nil {
		return &txMeta, errors.Wrapf(ErrBadInput, "txMeta: %v", err)
	}
	return &txMeta, nil
}

func decodeTransmitChecker(checkerMap MapParam) (txmgr.TransmitCheckerSpec, error) {
	var transmitChecker txmgr.TransmitCheckerSpec
	checkerDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &transmitChecker,
		ErrorUnused: true,
		DecodeHook: func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
			switch from {
			case stringType:
				switch to {
				case reflect.TypeOf(common.Address{}):
					ab, err := hex.DecodeString(data.(string))
					if err != nil {
						return nil, err
					}
					return common.BytesToAddress(ab), nil
				}
			}
			return data, nil
		},
	})
	if err != nil {
		return transmitChecker, errors.Wrapf(ErrBadInput, "transmitChecker: %v", err)
	}

	err = checkerDecoder.Decode(checkerMap)
	if err != nil {
		return transmitChecker, errors.Wrapf(ErrBadInput, "transmitChecker: %v", err)
	}
	return transmitChecker, nil
}

// txMeta is really only used for logging, so this is best-effort
func setJobIDOnMeta(lggr logger.Logger, vars Vars, meta *txmgr.TxMeta) {
	jobID, err := vars.Get("jobSpec.databaseID")
	if err != nil {
		return
	}
	switch v := jobID.(type) {
	case int64:
		vv := int32(v)
		meta.JobID = &vv
	default:
		logger.Sugared(lggr).AssumptionViolationf("expected type int32 for vars.jobSpec.databaseID; got: %T (value: %v)", jobID, jobID)
	}
}
