/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	// "bytes"
	"encoding/json"
	"fmt"
        "os"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// MarblesPrivateChaincode example Chaincode implementation
type MarblesPrivateChaincode struct {
}

type marble struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	RecordID   string `json:"recordid"`
	Owner	string `json:"owner"`    //the fieldtags are needed to keep case from bouncing around
	Sex string `json:"sex`
}

type marblePrivateDetails struct {
	ObjectType    string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	RecordID      string `json:"recordid"`    //the fieldtags are needed to keep case from bouncing around
	DataLabel     string `json:"datalabel"`
	Cholesterol   string `json:"cholesterol"`
	BloodPressure string `json:"bloodpressure"`
}

// Init initializes chaincode
// ===========================
func (t *MarblesPrivateChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *MarblesPrivateChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	switch function {
	case "initMarble":
		//create a new marble
		return t.initMarble(stub, args)
	case "readMarble":
		//read a marble
		return t.readMarble(stub, args)
	case "readMarblePrivateDetails":
		//read a marble private details
		return t.readMarblePrivateDetails(stub, args)
	case "transferMarble":
		//change owner of a specific marble
		return t.transferMarble(stub, args)
	case "delete":
		//delete a marble
		return t.delete(stub, args)
	// case "getMarblesByRange":
	// 	//get marbles based on range query
	// 	return t.getMarblesByRange(stub, args)
	// case "getMarbleHash":
	// 	// get private data hash for collectionMarbles
	// 	return t.getMarbleHash(stub, args)
	// case "getMarblePrivateDetailsHash":
	// 	// get private data hash for collectionMarblePrivateDetails
	// 	return t.getMarblePrivateDetailsHash(stub, args)
	default:
		//error
		fmt.Println("invoke did not find func: " + function)
		return shim.Error("Received unknown function invocation")
	}
}

// ============================================================
// initMarble - create a new marble, store into chaincode state
// ============================================================
func (t *MarblesPrivateChaincode) initMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	type marbleTransientInput struct {
		RecordID  string `json:"recordid"` //the fieldtags are needed to keep case from bouncing around
		Owner string `json:"owner"`
		DataLabel string `json:"datalabel"`
		Sex string `json:"sex`
		Cholesterol  string `json:"cholesterol"`
		BloodPressure string `json:"bloodpressure"`
	}

	// ==== Input sanitation ====
	fmt.Println("- start init marble")

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private marble data must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	marbleJsonBytes, ok := transMap["marble"]
	if !ok {
		return shim.Error("marble must be a key in the transient map")
	}

	if len(marbleJsonBytes) == 0 {
		return shim.Error("marble value in the transient map must be a non-empty JSON string")
	}

	var marbleInput marbleTransientInput
	err = json.Unmarshal(marbleJsonBytes, &marbleInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(marbleJsonBytes))
	}

	if len(marbleInput.RecordID) == 0 {
		return shim.Error("recordid field must be a non-empty string")
	}
	if len(marbleInput.DataLabel) == 0 {
		return shim.Error("datalabel field must be a non-empty string")
	}
	if len(marbleInput.Cholesterol) == 0 {
		return shim.Error("cholesterol must be a non-empty string")
	}
	if len(marbleInput.Owner) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}
	if len(marbleInput.BloodPressure) == 0 {
		return shim.Error("bloodpressure field must be a non-empty string")
	}

	// ==== Check if marble already exists ====
	marbleAsBytes, err := stub.GetPrivateData("collectionMarbles", marbleInput.RecordID)
	if err != nil {
		return shim.Error("Failed to get marble: " + err.Error())
	} else if marbleAsBytes != nil {
		fmt.Println("This marble already exists: " + marbleInput.RecordID)
		return shim.Error("This marble already exists: " + marbleInput.RecordID)
	}

	// ==== Create marble object and marshal to JSON ====
	marble := &marble{
		ObjectType: "marble",
		RecordID:	marbleInput.RecordID,
		Owner:	marbleInput.Owner,
		Sex: marbleInput.Sex,
	}
	marbleJSONasBytes, err := json.Marshal(marble)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save marble to state ===
	err = stub.PutPrivateData("collectionMarbles", marbleInput.RecordID, marbleJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// ==== Create marble private details object with price, marshal to JSON, and save to state ====
	marblePrivateDetails := &marblePrivateDetails{
		ObjectType: "marblePrivateDetails",
		RecordID:	marbleInput.RecordID,
		DataLabel:	marbleInput.DataLabel,
		Cholesterol:  marbleInput.Cholesterol,
		BloodPressure:	marbleInput.BloodPressure,
	}
	marblePrivateDetailsBytes, err := json.Marshal(marblePrivateDetails)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutPrivateData("collectionMarblePrivateDetails", marbleInput.RecordID, marblePrivateDetailsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//  ==== Index the marble to enable color-based range queries, e.g. return all blue marbles ====
	//  An 'index' is a normal key/value entry in state.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~datalabel~recordid.
	//  This will enable very efficient state range queries based on composite keys matching indexName~color~*
	indexName := "sex~recordid"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{marble.Sex, marble.RecordID})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the marble.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutPrivateData("collectionMarbles", colorNameIndexKey, value)

	// ==== Marble saved and indexed. Return success ====
	fmt.Println("- end init marble")
	return shim.Success(nil)
}

// ===============================================
// readMarble - read a marble from chaincode state
// ===============================================
func (t *MarblesPrivateChaincode) readMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateData("collectionMarbles", name) //get the marble from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Marble does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ===============================================
// readMarblereadMarblePrivateDetails - read a marble private details from chaincode state
// ===============================================
func (t *MarblesPrivateChaincode) readMarblePrivateDetails(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetPrivateData("collectionMarblePrivateDetails", name) //get the marble private details from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get private details for " + name + ": " + err.Error() + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Marble private details does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ===============================================
// getMarbleHash - get marble private data hash for collectionMarbles from chaincode state
// ===============================================
// func (t *MarblesPrivateChaincode) getMarbleHash(stub shim.ChaincodeStubInterface, args []string) pb.Response {
// 	var name, jsonResp string
// 	var err error

// 	if len(args) != 1 {
// 		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
// 	}

// 	name = args[0]
// 	valAsbytes, err := stub.GetPrivateDataHash("collectionMarbles", name)
// 	if err != nil {
// 		jsonResp = "{\"Error\":\"Failed to get marble private data hash for " + name + "\"}"
// 		return shim.Error(jsonResp)
// 	} else if valAsbytes == nil {
// 		jsonResp = "{\"Error\":\"Marble private marble data hash does not exist: " + name + "\"}"
// 		return shim.Error(jsonResp)
// 	}

// 	return shim.Success(valAsbytes)
// }

// ===============================================
// getMarblePrivateDetailsHash - get marble private data hash for collectionMarblePrivateDetails from chaincode state
// ===============================================
// func (t *MarblesPrivateChaincode) getMarblePrivateDetailsHash(stub shim.ChaincodeStubInterface, args []string) pb.Response {
// 	var name, jsonResp string
// 	var err error

// 	if len(args) != 1 {
// 		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
// 	}

// 	name = args[0]
// 	valAsbytes, err := stub.GetPrivateDataHash("collectionMarblePrivateDetails", name)
// 	if err != nil {
// 		jsonResp = "{\"Error\":\"Failed to get marble private details hash for " + name + ": " + err.Error() + "\"}"
// 		return shim.Error(jsonResp)
// 	} else if valAsbytes == nil {
// 		jsonResp = "{\"Error\":\"Marble private details hash does not exist: " + name + "\"}"
// 		return shim.Error(jsonResp)
// 	}

// 	return shim.Success(valAsbytes)
// }

// ==================================================
// delete - remove a marble key/value pair from state
// ==================================================
func (t *MarblesPrivateChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Println("- start delete marble")

	type marbleDeleteTransientInput struct {
		RecordID string `json:"recordid"`
	}

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private marble name must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	marbleDeleteJsonBytes, ok := transMap["marble_delete"]
	if !ok {
		return shim.Error("marble_delete must be a key in the transient map")
	}

	if len(marbleDeleteJsonBytes) == 0 {
		return shim.Error("marble_delete value in the transient map must be a non-empty JSON string")
	}

	var marbleDeleteInput marbleDeleteTransientInput
	err = json.Unmarshal(marbleDeleteJsonBytes, &marbleDeleteInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(marbleDeleteJsonBytes))
	}

	if len(marbleDeleteInput.RecordID) == 0 {
		return shim.Error("recordid field must be a non-empty string")
	}

	// to maintain the datalabel~recordid index, we need to read the marble first and get its color
	valAsbytes, err := stub.GetPrivateData("collectionMarbles", marbleDeleteInput.RecordID) //get the marble from chaincode state
	if err != nil {
		return shim.Error("Failed to get state for " + marbleDeleteInput.RecordID)
	} else if valAsbytes == nil {
		return shim.Error("Marble does not exist: " + marbleDeleteInput.RecordID)
	}

	var marbleToDelete marble
	err = json.Unmarshal([]byte(valAsbytes), &marbleToDelete)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(valAsbytes))
	}

	// delete the marble from state
	err = stub.DelPrivateData("collectionMarbles", marbleDeleteInput.RecordID)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// Also delete the marble from the datalabel~recordid index
	indexName := "sex~recordid"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{marbleToDelete.Sex, marbleToDelete.RecordID})
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.DelPrivateData("collectionMarbles", colorNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// Finally, delete private details of marble
	err = stub.DelPrivateData("collectionMarblePrivateDetails", marbleDeleteInput.RecordID)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ===========================================================
// transfer a marble by setting a new owner name on the marble
// ===========================================================
func (t *MarblesPrivateChaincode) transferMarble(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	fmt.Println("- start transfer marble")

	type marbleTransferTransientInput struct {
		RecordID  string `json:"recordid"`
		Owner string `json:"owner"`
	}

	if len(args) != 0 {
		return shim.Error("Incorrect number of arguments. Private marble data must be passed in transient map.")
	}

	transMap, err := stub.GetTransient()
	if err != nil {
		return shim.Error("Error getting transient: " + err.Error())
	}

	marbleOwnerJsonBytes, ok := transMap["marble_owner"]
	if !ok {
		return shim.Error("marble_owner must be a key in the transient map")
	}

	if len(marbleOwnerJsonBytes) == 0 {
		return shim.Error("marble_owner value in the transient map must be a non-empty JSON string")
	}

	var marbleTransferInput marbleTransferTransientInput
	err = json.Unmarshal(marbleOwnerJsonBytes, &marbleTransferInput)
	if err != nil {
		return shim.Error("Failed to decode JSON of: " + string(marbleOwnerJsonBytes))
	}

	if len(marbleTransferInput.RecordID) == 0 {
		return shim.Error("recordid field must be a non-empty string")
	}
	if len(marbleTransferInput.Owner) == 0 {
		return shim.Error("owner field must be a non-empty string")
	}

	marbleAsBytes, err := stub.GetPrivateData("collectionMarbles", marbleTransferInput.RecordID)
	if err != nil {
		return shim.Error("Failed to get marble:" + err.Error())
	} else if marbleAsBytes == nil {
		return shim.Error("Marble does not exist: " + marbleTransferInput.RecordID)
	}

	marbleToTransfer := marble{}
	err = json.Unmarshal(marbleAsBytes, &marbleToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	marbleToTransfer.Owner = marbleTransferInput.Owner //change the owner

	marbleJSONasBytes, _ := json.Marshal(marbleToTransfer)
	err = stub.PutPrivateData("collectionMarbles", marbleToTransfer.RecordID, marbleJSONasBytes) //rewrite the marble
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end transferMarble (success)")
	return shim.Success(nil)
}

// ===========================================================================================
// getMarblesByRange performs a range query based on the start and end keys provided.

// Read-only function results are not typically submitted to ordering. If the read-only
// results are submitted to ordering, or if the query is used in an update transaction
// and submitted to ordering, then the committing peers will re-execute to guarantee that
// result sets are stable between endorsement time and commit time. The transaction is
// invalidated by the committing peers if the result set has changed between endorsement
// time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
// ===========================================================================================
// func (t *MarblesPrivateChaincode) getMarblesByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

// 	if len(args) < 2 {
// 		return shim.Error("Incorrect number of arguments. Expecting 2")
// 	}

// 	startKey := args[0]
// 	endKey := args[1]

// 	resultsIterator, err := stub.GetPrivateDataByRange("collectionMarbles", startKey, endKey)
// 	if err != nil {
// 		return shim.Error(err.Error())
// 	}
// 	defer resultsIterator.Close()

// 	// buffer is a JSON array containing QueryResults
// 	var buffer bytes.Buffer
// 	buffer.WriteString("[")

// 	bArrayMemberAlreadyWritten := false
// 	for resultsIterator.HasNext() {
// 		queryResponse, err := resultsIterator.Next()
// 		if err != nil {
// 			return shim.Error(err.Error())
// 		}
// 		// Add a comma before array members, suppress it for the first array member
// 		if bArrayMemberAlreadyWritten {
// 			buffer.WriteString(",")
// 		}

// 		buffer.WriteString(
// 			fmt.Sprintf(
// 				`{"Key":"%s", "Record":%s}`,
// 				queryResponse.Key, queryResponse.Value,
// 			),
// 		)
// 		bArrayMemberAlreadyWritten = true
// 	}
// 	buffer.WriteString("]")

// 	fmt.Printf("- getMarblesByRange queryResult:\n%s\n", buffer.String())

// 	return shim.Success(buffer.Bytes())
// }

func main() {
	err := shim.Start(&MarblesPrivateChaincode{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Exiting Simple chaincode: %s", err)
		os.Exit(2)
	}
}
