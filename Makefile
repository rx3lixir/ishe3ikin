# Binary output name
BINARY_NAME=isheikin

# Default make command
all: build

# Build the binary
build:
	@echo "Building..."
	go build -o ./bin/$(BINARY_NAME) ./cmd/isheikin/

# Run the server
run: build
	@echo "Running..."
	./bin/$(BINARY_NAME)

# Test your application
test:
	@echo "Testing..."
	go test ./...

# Clean up binaries
clean:
	@echo "Cleaning..."
	go clean
	rm -f ./bin/$(BINARY_NAME)

# Help command to display available commands
help:
	@echo "Makefile commands:"
	@echo "all    - Build the application"
	@echo "build  - Build the binary"
	@echo "run    - Build and run the application"
	@echo "test   - Run tests"
	@echo "clean  - Remove binaries"
	@echo "help   - Display this help"

# Mark commands that don't correspond to files as .PHONY
.PHONY: all build run test clean help
