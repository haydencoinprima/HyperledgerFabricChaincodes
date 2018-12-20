package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	// "strings"
	 "time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type WalletChainCode struct {

}

type transaction struct {
	Id int64 `json:"id"`
	FromAddr string `json:"fromAddr"`
	ToAddr string `json:"toAddr"`
	TxType string `json:"txType"` //deposit, withdraw, maker, taker
	AssetName string `json:"assetName"` //btc, eth
	Qty float64 `json:"qty"`
	FeesName string `json:"feesName"` //btc, eth
	Fees float64 `json:"fees"`
	Timestamp string `json:"timestamp"`
}

type wallet struct {
	WalletId string `json:"walletId"` //format: {asset name}_{client id}
	AssetName string `json:"assetName"`
	Available float64 `json:"available"`
	InOrder float64 `json:"inOrder"`
	DepositFees float64 `json:"depositFees"`
	WithdrawFees float64 `json:"withdrawFees"`
	TakerFees float64 `json:"takerFees"`
	MakerFees float64 `json:"makerFees"`
	Tx transaction `json:"tx"` //store the transaction
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	err := shim.Start(new(WalletChainCode))
	if err != nil {
		fmt.Printf("Error starting Wallet chaincode: %s", err)
	}
}

func (t *WalletChainCode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *WalletChainCode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "query" {
		assetName := args[0]
		clientId := args[1]
		return t.Query(stub, assetName, clientId)
	} else if function == "getHistory" {
			assetName := args[0]
			clientId := args[1]
			return t.GetHistory(stub, assetName, clientId)
	} else if function == "deposit" {
		assetName := args[0]
		clientId := args[1]
		amount, _ := strconv.ParseFloat(args[2], 64)
		fees, _ := strconv.ParseFloat(args[3], 64)
		return t.Deposit(stub, assetName, clientId, amount, fees)
	} else if function == "withdraw" {
		assetName := args[0]
		clientId := args[1]
		amount, _ := strconv.ParseFloat(args[2], 64)
		fees, _ := strconv.ParseFloat(args[3], 64)
		return t.Withdraw(stub, assetName, clientId, amount, fees)
	} 
	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

func NewWallet(assetName string, clientId string) wallet {
	w := wallet{}
	w.AssetName = assetName
	w.WalletId = assetName + "_" + clientId
	w.Available = 0.0
	w.InOrder = 0.0
	w.MakerFees = 0.0
	w.TakerFees = 0.0
	w.DepositFees = 0.0
	w.WithdrawFees = 0.0
	tx := transaction{}
	tx.Id = 0
	w.Tx = tx
	return w
}

func (t *WalletChainCode) Query(stub shim.ChaincodeStubInterface, assetName string, clientId string) pb.Response {
	key := assetName + "_" + clientId
	walletBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to get state for " + key + ", msg: " + err.Error() + "\"}")
	} else if walletBytes == nil {
		walletJSON := NewWallet(assetName, clientId)
		walletBytes, err := json.Marshal(walletJSON)
		if err != nil {
			return shim.Error("{\"Error\":\"Failed to encode for " + key + ", msg: " + err.Error() + "\"}")
		}
		err = stub.PutState(key, walletBytes)
		if err != nil {
			return shim.Error("{\"Error\":\"Failed to put state for " + key + ", msg: " + err.Error() + "\"}")
		} 
		return shim.Success(walletBytes)
	} else {
		return shim.Success(walletBytes)
	}	
}

func (t *WalletChainCode) Deposit(stub shim.ChaincodeStubInterface, assetName string, clientId string, amount float64, fees float64) pb.Response {
	var walletJSON wallet

	if amount <= 0 || fees < 0 || amount - fees < 0 {
		return shim.Error("{\"Error\":\"Failed to deposit, amount is " + fmt.Sprintf("%f", amount) + ", fees is " + fmt.Sprintf("%f", fees) + "\"}")
	}

	key := assetName + "_" + clientId
	walletBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to get state for " + key + "\"}")
	} else if walletBytes == nil {
		walletJSON = NewWallet(assetName, clientId)
	} else {
		err = json.Unmarshal([]byte(walletBytes), &walletJSON) 
		if err != nil {
			return shim.Error("{\"Error\":\"Failed to decode JSON of " + key + "\"}")
		}
	}
		
	walletJSON.Available += (amount - fees)
	walletJSON.DepositFees += fees	
	
	tx := transaction{}
	tx.Id = walletJSON.Tx.Id + 1
	tx.FromAddr = ""
	tx.ToAddr = ""
	tx.TxType = "deposit"
	tx.AssetName = assetName
	tx.Qty = amount
	tx.FeesName = assetName
	tx.Fees = fees
	tx.Timestamp = time.Now().String()

	walletJSON.Tx = tx
	walletBytes, err = json.Marshal(walletJSON)	
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to encode JSON of " + key + "\"}")
	}

	err = stub.PutState(key, walletBytes)
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to put state of " + key + "\"}")
	}

	return shim.Success(walletBytes)
}

func (t *WalletChainCode) Withdraw(stub shim.ChaincodeStubInterface, assetName string, clientId string, amount float64, fees float64) pb.Response {
	var walletJSON wallet
	key := assetName + "_" + clientId

	if amount <= 0 || fees < 0 || amount - fees < 0 {
		return shim.Error("{\"Error\":\"Failed to withdraw for " + key + ", negative value, amount is " + fmt.Sprintf("%f", amount) + ", fees is " + fmt.Sprintf("%f", fees) + "\"}")
	}

	
	walletBytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to get state for " + key + "\"}")
	} else if walletBytes == nil {
		walletJSON = NewWallet(assetName, clientId)
	} else {
		err = json.Unmarshal([]byte(walletBytes), &walletJSON) 
		if err != nil {
			return shim.Error("{\"Error\":\"Failed to decode JSON of " + key + "\"}")
		}
	}

	available := walletJSON.Available
	if available <= 0 || available - amount - fees < 0 {
		return shim.Error("{\"Error\":\"Failed to withdraw for " + key + ", not enough, amount is " + fmt.Sprintf("%f", amount) + ", available is " + fmt.Sprintf("%f", available) + "\"}")
	}

	walletJSON.Available -= (amount + fees)
	walletJSON.WithdrawFees += fees	
	
	tx := transaction{}
	tx.Id = walletJSON.Tx.Id + 1
	tx.FromAddr = ""
	tx.ToAddr = ""
	tx.TxType = "withdraw"
	tx.AssetName = assetName
	tx.Qty = amount
	tx.FeesName = assetName
	tx.Fees = fees
	tx.Timestamp = time.Now().String()

	walletJSON.Tx = tx
	walletBytes, err = json.Marshal(walletJSON)	
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to encode JSON of " + key + "\"}")
	}

	err = stub.PutState(key, walletBytes)
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to put state of " + key + "\"}")
	}

	return shim.Success(walletBytes)
}

func (t *WalletChainCode) GetHistory(stub shim.ChaincodeStubInterface, assetName string, clientId string) pb.Response {
	key := assetName + "_" + clientId
	iter, err := stub.GetHistoryForKey(key)
	if err != nil {
		return shim.Error("{\"Error\":\"Failed to get history for " + key + ", error: " + err.Error()+ "\"}")
	}
	defer iter.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")
	for iter.HasNext() {
		item, err := iter.Next()
		if err != nil {
			return shim.Error("{\"Error\":\"Failed to iterate history for " + key + ", error: " + err.Error() + "\"}")
		}
		buffer.WriteString(string(item.Value))
		if iter.HasNext() {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("]")
	return shim.Success(buffer.Bytes())
}