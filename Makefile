proto:
	@echo == Generating protobuf code ==
	protoc --go_out=. model/*.proto
	protoc --go_out=. network/model/*.proto
	@echo
