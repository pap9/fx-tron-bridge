package client

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethAbi "github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	sdkCommon "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	sdkContract "github.com/fbsobreira/gotron-sdk/pkg/proto/core/contract"
	"github.com/functionx/fx-tron-bridge/contract"
)

var fxBridgeAbi ethAbi.ABI

func init() {
	fxBridgeLogicAbi, err := ethAbi.JSON(strings.NewReader(contract.FxBridgeTronMetaData.ABI))
	if err != nil {
		panic("contract abi json format error")
	}
	fxBridgeAbi = fxBridgeLogicAbi
}

func unpackLog(abi ethAbi.ABI, out interface{}, event string, log types.Log) error {
	if log.Topics[0] != abi.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed ethAbi.Arguments
	for _, arg := range abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return ethAbi.ParseTopics(out, indexed, log.Topics[1:])
}

func (c *TronClient) QueryBlockEvent(contractAddress string, blockNumber uint64) (
	[]*contract.FxBridgeTronSendToFxEvent, []*contract.FxBridgeTronTransactionBatchExecutedEvent, []*contract.FxBridgeTronAddBridgeTokenEvent, []*contract.FxBridgeTronOracleSetUpdatedEvent, error,
) {

	sendToFxEvents := make([]*contract.FxBridgeTronSendToFxEvent, 0)
	transactionBatchExecutedEvents := make([]*contract.FxBridgeTronTransactionBatchExecutedEvent, 0)
	addBridgeTokenEvents := make([]*contract.FxBridgeTronAddBridgeTokenEvent, 0)
	oracleSetUpdatedEvents := make([]*contract.FxBridgeTronOracleSetUpdatedEvent, 0)

	blockInfo, err := c.GetBlockInfoByNum(int64(blockNumber))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	for _, transactionInfo := range blockInfo.TransactionInfo {
		for _, sdkLog := range transactionInfo.Log {
			if core.Transaction_Result_SUCCESS != transactionInfo.Receipt.Result || len(sdkLog.Topics) <= 0 {
				continue
			}
			if contractAddress != sdkCommon.EncodeCheck(transactionInfo.ContractAddress) && contractAddress != sdkCommon.EncodeCheck(append([]byte{address.TronBytePrefix}, sdkLog.Address...)) {
				continue
			}
			topics := make([]ethCommon.Hash, len(sdkLog.Topics))
			for logIndex, topic := range sdkLog.Topics {
				topics[logIndex] = ethCommon.BytesToHash(topic)
			}
			log := types.Log{Topics: topics, Data: sdkLog.Data, TxHash: ethCommon.BytesToHash(transactionInfo.Id)}

			switch ethCommon.BytesToHash(sdkLog.Topics[0]).Hex() {
			case fxBridgeAbi.Events["SendToFxEvent"].ID.String():
				bridgeLogicSendToFxEvent := new(contract.FxBridgeTronSendToFxEvent)
				if err := unpackLog(fxBridgeAbi, bridgeLogicSendToFxEvent, "SendToFxEvent", log); err != nil {
					return nil, nil, nil, nil, err
				}
				bridgeLogicSendToFxEvent.Raw = log
				sendToFxEvents = append(sendToFxEvents, bridgeLogicSendToFxEvent)
			case fxBridgeAbi.Events["TransactionBatchExecutedEvent"].ID.String():
				bridgeLogicTransactionBatchExecutedEvent := new(contract.FxBridgeTronTransactionBatchExecutedEvent)
				if err := unpackLog(fxBridgeAbi, bridgeLogicTransactionBatchExecutedEvent, "TransactionBatchExecutedEvent", log); err != nil {
					return nil, nil, nil, nil, err
				}
				bridgeLogicTransactionBatchExecutedEvent.Raw = log
				transactionBatchExecutedEvents = append(transactionBatchExecutedEvents, bridgeLogicTransactionBatchExecutedEvent)
			case fxBridgeAbi.Events["AddBridgeTokenEvent"].ID.String():
				bridgeLogicAddBridgeTokenEvent := new(contract.FxBridgeTronAddBridgeTokenEvent)
				if err := unpackLog(fxBridgeAbi, bridgeLogicAddBridgeTokenEvent, "AddBridgeTokenEvent", log); err != nil {
					return nil, nil, nil, nil, err
				}
				bridgeLogicAddBridgeTokenEvent.Raw = log
				addBridgeTokenEvents = append(addBridgeTokenEvents, bridgeLogicAddBridgeTokenEvent)
			case fxBridgeAbi.Events["OracleSetUpdatedEvent"].ID.String():
				bridgeLogicOracleSetUpdatedEvent := new(contract.FxBridgeTronOracleSetUpdatedEvent)
				if err := unpackLog(fxBridgeAbi, bridgeLogicOracleSetUpdatedEvent, "OracleSetUpdatedEvent", log); err != nil {
					return nil, nil, nil, nil, err
				}
				bridgeLogicOracleSetUpdatedEvent.Raw = log
				oracleSetUpdatedEvents = append(oracleSetUpdatedEvents, bridgeLogicOracleSetUpdatedEvent)
			}
		}
	}

	return sendToFxEvents, transactionBatchExecutedEvents, addBridgeTokenEvents, oracleSetUpdatedEvents, nil
}

func (c *TronClient) QueryOracleSetUpdatedEvent(contractAddress string, blockNumber uint64) ([]*contract.FxBridgeTronOracleSetUpdatedEvent, error) {
	oracleSetUpdatedEvents := make([]*contract.FxBridgeTronOracleSetUpdatedEvent, 0)

	blockInfo, err := c.GetBlockInfoByNum(int64(blockNumber))
	if err != nil {
		return nil, err
	}

	for _, transactionInfo := range blockInfo.TransactionInfo {
		for _, sdkLog := range transactionInfo.Log {
			if core.Transaction_Result_SUCCESS != transactionInfo.Receipt.Result || len(sdkLog.Topics) <= 0 {
				continue
			}
			if contractAddress != sdkCommon.EncodeCheck(transactionInfo.ContractAddress) && contractAddress != sdkCommon.EncodeCheck(append([]byte{address.TronBytePrefix}, sdkLog.Address...)) {
				continue
			}

			if ethCommon.BytesToHash(sdkLog.Topics[0]).Hex() != fxBridgeAbi.Events["OracleSetUpdatedEvent"].ID.String() {
				continue
			}

			topics := make([]ethCommon.Hash, len(sdkLog.Topics))
			for logIndex, topic := range sdkLog.Topics {
				topics[logIndex] = ethCommon.BytesToHash(topic)
			}
			log := types.Log{Topics: topics, Data: sdkLog.Data, TxHash: ethCommon.BytesToHash(transactionInfo.Id)}

			bridgeLogicOracleSetUpdatedEvent := new(contract.FxBridgeTronOracleSetUpdatedEvent)
			if err := unpackLog(fxBridgeAbi, bridgeLogicOracleSetUpdatedEvent, "OracleSetUpdatedEvent", log); err != nil {
				return nil, err
			}
			bridgeLogicOracleSetUpdatedEvent.Raw = log
			oracleSetUpdatedEvents = append(oracleSetUpdatedEvents, bridgeLogicOracleSetUpdatedEvent)
		}
	}

	return oracleSetUpdatedEvents, nil
}

func (c *TronClient) StateLastOracleSetNonce(contractAddress string) (uint64, error) {
	transactionExtention, err := c.TriggerConstantContract("", contractAddress, "state_lastOracleSetNonce()", "")
	if err != nil {
		return 0, err
	}
	if len(transactionExtention.ConstantResult) <= 0 {
		return 0, fmt.Errorf("trigger constant state_lastOracleSetNonce error contractAddress: %v", contractAddress)
	}
	return new(big.Int).SetBytes(transactionExtention.ConstantResult[0]).Uint64(), nil
}

func (c *TronClient) StateFxBridgeId(contractAddress string) (string, error) {
	transactionExtention, err := c.TriggerConstantContract("", contractAddress, "state_fxBridgeId()", "")
	if err != nil {
		return "", err
	}
	if len(transactionExtention.ConstantResult) <= 0 {
		return "", fmt.Errorf("trigger constant state_fxBridgeId error contractAddress: %v", contractAddress)
	}
	bridgeId := transactionExtention.ConstantResult[0]
	var length = len(bridgeId) - 1
	for length > 0 && bridgeId[length-1] == 0 {
		length--
	}
	return string(bridgeId[:length]), nil
}

func (c *TronClient) StateLastOracleSetHeight(contractAddress string) (uint64, error) {
	transactionExtention, err := c.TriggerConstantContract("", contractAddress, "state_laseOracleSetHeight()", "")
	if err != nil {
		return 0, err
	}
	if len(transactionExtention.ConstantResult) <= 0 {
		return 0, fmt.Errorf("trigger constant state_lastOracleSetHeight error contractAddress: %v", contractAddress)
	}
	return new(big.Int).SetBytes(transactionExtention.ConstantResult[0]).Uint64(), nil
}

func (c *TronClient) LastBatchNonce(contractAddress string, erc20Address string) (uint64, error) {
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return 0, err
	}
	params := []abi.Param{
		{"address": erc20Address},
	}
	data, err := abi.Pack("lastBatchNonce(address)", params)
	if err != nil {
		return 0, err
	}
	tx := &sdkContract.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            data,
	}
	transactionExtention, err := c.Client.TriggerConstantContract(context.Background(), tx)
	if err != nil {
		return 0, err
	}

	return new(big.Int).SetBytes(transactionExtention.ConstantResult[0]).Uint64(), nil
}

func (c *TronClient) Allowance(contractAddress, owner, spender string) (*big.Int, error) {
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	params := []abi.Param{
		{"address": owner},
		{"address": spender},
	}
	data, err := abi.Pack("allowance(address,address)", params)
	if err != nil {
		return nil, err
	}
	tx := &sdkContract.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            data,
	}
	transactionExtention, err := c.Client.TriggerConstantContract(context.Background(), tx)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(transactionExtention.ConstantResult[0]), nil
}

func (c *TronClient) GetTokenStatus(contractAddress, tokenAddress string) (bool, bool, bool, error) {
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return false, false, false, err
	}
	params := []abi.Param{
		{"address": tokenAddress},
	}
	data, err := abi.Pack("tokenStatus(address)", params)
	if err != nil {
		return false, false, false, err
	}
	tx := &sdkContract.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            data,
	}
	transactionExtention, err := c.Client.TriggerConstantContract(context.Background(), tx)
	if err != nil {
		return false, false, false, err
	}

	outputMap := make(map[string]interface{})
	err = fxBridgeAbi.UnpackIntoMap(outputMap, "tokenStatus", transactionExtention.ConstantResult[0])
	if err != nil {
		return false, false, false, err
	}

	return outputMap["isOriginated"].(bool), outputMap["isActive"].(bool), outputMap["isExist"].(bool), nil
}

func (c *TronClient) GetBridgeTokenList(contractAddress string) ([]contract.FxBridgeToken, error) {
	transactionExtention, err := c.TriggerConstantContract("", contractAddress, "getBridgeTokenList()", "")
	if err != nil {
		return nil, err
	}
	unpackOut, err := fxBridgeAbi.Unpack("getBridgeTokenList", transactionExtention.ConstantResult[0])
	if err != nil {
		return nil, err
	}

	return *ethAbi.ConvertType(unpackOut[0], new([]contract.FxBridgeToken)).(*[]contract.FxBridgeToken), nil
}
