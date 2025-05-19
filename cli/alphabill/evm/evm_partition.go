package evm

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strconv"

	util "github.com/ipfs/go-ipfs-util"

	evmsdk "github.com/alphabill-org/alphabill-evm/txsystem/evm"
	"github.com/alphabill-org/alphabill-go-base/types"
	"github.com/alphabill-org/alphabill/cli/alphabill/cmd"
	"github.com/alphabill-org/alphabill/partition"
	"github.com/alphabill-org/alphabill/state"
	"github.com/alphabill-org/alphabill/txsystem"

	"github.com/alphabill-org/alphabill-evm/internal/txsystem/evm"
	"github.com/alphabill-org/alphabill-evm/internal/txsystem/evm/api"
)

const (
	evmBlockGasLimit = "blockGasLimit"
	evmGasUnitPrice  = "gasUnitPrice"
)

type (
	EvmPartition struct {
		partitionTypeID types.PartitionTypeID
	}

	EvmPartitionParams struct {
		BlockGasLimit uint64 // max units of gas processed in each block
		GasUnitPrice  uint64 // gas unit price in wei
	}
)

func NewEvmPartition() *EvmPartition {
	return &EvmPartition{
		partitionTypeID: evmsdk.PartitionTypeID,
	}
}
func (p *EvmPartition) PartitionTypeID() types.PartitionTypeID {
	return p.partitionTypeID
}

func (p *EvmPartition) PartitionTypeIDString() string {
	return "evm"
}

func (p *EvmPartition) DefaultPartitionParams(flags *cmd.ShardConfGenerateFlags) map[string]string {
	partitionParams := make(map[string]string, 1)

	partitionParams[evmGasUnitPrice] = strconv.FormatUint(evm.DefaultGasPrice, 10)
	partitionParams[evmBlockGasLimit] = strconv.FormatUint(evm.DefaultBlockGasLimit, 10)

	return partitionParams
}

func (p *EvmPartition) NewGenesisState(pdr *types.PartitionDescriptionRecord) (*state.State, error) {
	return state.NewEmptyState(), nil
}

func (p *EvmPartition) CreateTxSystem(flags *cmd.ShardNodeRunFlags, nodeConf *partition.NodeConf) (txsystem.TransactionSystem, error) {
	stateFilePath := flags.PathWithDefault(flags.StateFile, cmd.StateFileName)
	state, header, err := loadStateFile(stateFilePath, evm.NewUnitData)
	if err != nil {
		return nil, fmt.Errorf("failed to load state file: %w", err)
	}
	params, err := ParseEvmPartitionParams(nodeConf.ShardConf())
	if err != nil {
		return nil, fmt.Errorf("failed to validate evm partition params: %w", err)
	}
	txs, err := evm.NewEVMTxSystem(
		nodeConf.ShardConf().NetworkID,
		nodeConf.ShardConf().PartitionID,
		nodeConf.Observability(),
		evm.WithBlockGasLimit(params.BlockGasLimit),
		evm.WithGasPrice(params.GasUnitPrice),
		evm.WithBlockDB(nodeConf.BlockStore()),
		evm.WithTrustBase(nodeConf.TrustBase()),
		evm.WithState(state),
		evm.WithExecutedTransactions(header.ExecutedTransactions),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create evm tx system: %w", err)
	}

	flags.ServerConfiguration.Router = api.NewAPI(
		state,
		nodeConf.ShardConf().PartitionID,
		big.NewInt(0).SetUint64(params.BlockGasLimit),
		params.GasUnitPrice,
		nodeConf.Observability().Logger(),
	)
	return txs, nil
}

func ParseEvmPartitionParams(shardConf *types.PartitionDescriptionRecord) (*EvmPartitionParams, error) {
	var params EvmPartitionParams
	for key, value := range shardConf.PartitionParams {
		switch key {
		case evmBlockGasLimit:
			parsedValue, err := parseUint64(key, value)
			if err != nil {
				return nil, err
			}
			params.BlockGasLimit = parsedValue
		case evmGasUnitPrice:
			parsedValue, err := parseUint64(key, value)
			if err != nil {
				return nil, err
			}
			if parsedValue > math.MaxInt64 {
				return nil, fmt.Errorf("invalid gasUnitPrice, max value is %v", math.MaxInt64)
			}
			params.GasUnitPrice = parsedValue
		default:
			return nil, fmt.Errorf("unexpected partition param: %s", key)
		}
	}
	return &params, nil
}

func parseUint64(key, value string) (uint64, error) {
	ret, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse param %q value: %w", key, err)
	}
	return ret, nil
}

func loadStateFile(stateFilePath string, unitDataConstructor state.UnitDataConstructor) (*state.State, *state.Header, error) {
	if !util.FileExists(stateFilePath) {
		return nil, nil, fmt.Errorf("state file '%s' not found", stateFilePath)
	}

	stateFile, err := os.Open(filepath.Clean(stateFilePath))
	if err != nil {
		return nil, nil, err
	}
	defer stateFile.Close()

	s, header, err := state.NewRecoveredState(stateFile, unitDataConstructor)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build state tree from state file: %w", err)
	}
	return s, header, nil
}
