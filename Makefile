gen:
	protoc --go_out=plugins=grpc:. distcachepb/*.proto
