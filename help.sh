mkdir -p vars/chaincode/healthcare/go
cp -R healthcare/healthcare_collection_config.json vars/healthcare_collection_config.json
cp -R chaincode/privatemarbles/go/main.go vars/chaincode/healthcare/go/main.go
cp -R chaincode/privatemarbles/go/go.mod vars/chaincode/healthcare/go/go.mod