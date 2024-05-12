
PROTO_DIR := proto
GEN_DIR := gen/go
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
GENERATED_FILES := $(PROTO_FILES:$(PROTO_DIR)/%.proto=$(GEN_DIR)/example/%.pb.go)


all: proto

proto: $(GENERATED_FILES)

$(GEN_DIR)/example/%.pb.go: $(PROTO_DIR)/%.proto
	buf generate


clean:
	rm -rf $(GEN_DIR)
	rm -rf test

.PHONY: all proto clean