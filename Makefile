
PROTO_DIR := proto
GEN_DIR := gen/go
PROTO_FILES := $(wildcard $(PROTO_DIR)/rr/*.proto)
GENERATED_FILES := $(PROTO_FILES:$(PROTO_DIR)/%.proto=$(GEN_DIR)/rr/%.pb.go)

all: proto

proto: $(GENERATED_FILES)


$(GEN_DIR)/rr/%.pb.go: $(PROTO_DIR)/%.proto
	buf generate


clean:
	rm -rf $(GEN_DIR)
	rm -rf test

.PHONY: all proto clean
