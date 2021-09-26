protoutil:
    protoc --go_out=. -I gateway/protoutil common.proto
    protoc --go_out=. -I gateway/protoutil channel.proto
    protoc --go_out=. -I gateway/protoutil chaincode.proto
    protoc --go_out=. -I gateway/protoutil proposal.proto