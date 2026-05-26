# Simplified Docker Setup for Thread Management with RAG

## 1. Docker Compose (`docker-compose.yml`)

```yaml
version: '3.8'

services:
  # Cosmos DB Emulator
  cosmos-emulator:
    image: mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator:latest
    container_name: cosmos-emulator
    ports:
      - "8081:8081"
      - "10250-10255:10250-10255"
    environment:
      - AZURE_COSMOS_EMULATOR_PARTITION_COUNT=10
      - AZURE_COSMOS_EMULATOR_ENABLE_DATA_PERSISTENCE=true
    volumes:
      - cosmos-data:/data
    networks:
      - deepenc-network

  # Qdrant Vector Database
  qdrant:
    image: qdrant/qdrant:latest
    container_name: qdrant
    ports:
      - "6333:6333"
    volumes:
      - qdrant-data:/qdrant/storage
    environment:
      - QDRANT__LOG_LEVEL=INFO
    networks:
      - deepenc-network

  # Backend Application
  deepenc-backend:
    build: .
    container_name: deepenc-backend
    ports:
      - "8080:8080"
    environment:
      # Cosmos DB
      - COSMOS_ENDPOINT=https://cosmos-emulator:8081
      - COSMOS_EMULATOR_INSECURE=true
      - COSMOS_KEY=C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==
      - COSMOS_DB=deepenc
      - COSMOS_CONTAINER=messages
      - COSMOS_PARTITION_KEY=/threadId

      # Vector Store
      - QDRANT_URL=http://qdrant:6333
      - EMBEDDING_MODEL=text-embedding-3-small
    env_file:
      - .env
    depends_on:
      - cosmos-emulator
      - qdrant
    networks:
      - deepenc-network

volumes:
  cosmos-data:
  qdrant-data:

networks:
  deepenc-network:
    driver: bridge
```

## 2. Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/healthz || exit 1

CMD ["./main"]
```

## 3. Go Module Files

### `go.mod`
```go
module deepenc-backend

go 1.21

require (
    github.com/Azure/azure-sdk-for-go/sdk/azcore v1.9.1
    github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos v1.0.1
    github.com/google/uuid v1.5.0
)

require (
    github.com/Azure/azure-sdk-for-go/sdk/internal v1.5.1 // indirect
    golang.org/x/net v0.19.0 // indirect
    golang.org/x/text v0.14.0 // indirect
)
```

### `go.sum`
```
github.com/Azure/azure-sdk-for-go/sdk/azcore v1.9.1 h1:lGlwhPtrX6EVml1hO0ivjkUxsSyl4dsiw9qcA1k/3IQ=
github.com/Azure/azure-sdk-for-go/sdk/azcore v1.9.1/go.mod h1:RKUqNu35KJYcVG/fqTRqmuXJZYNhYkBrnC/hX7yGbTA=
github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos v1.0.1 h1:Mf44k5zoNL0iR/eJKjt5WT5dVPR6/jqW7s8IRzB0k50=
github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos v1.0.1/go.mod h1:HiYnz0ld3kRAJ5oJPI0hHiVN2yrnvdTnhEPYPB+rXHQ=
github.com/Azure/azure-sdk-for-go/sdk/internal v1.5.1 h1:6oNBlSdi1QqM1PNW7FPA6xOGA5UNsXnkaYZz9vdPGhA=
github.com/Azure/azure-sdk-for-go/sdk/internal v1.5.1/go.mod h1:s4kgfzA0covAXNicZHDMN58jExvcng2mC/DepXiF1EI=
github.com/google/uuid v1.5.0 h1:1p67kYwdtXjb0gL0BPiP1Av9wiZPo5A8z2cWkTZ+eyU=
github.com/google/uuid v1.5.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
golang.org/x/net v0.19.0 h1:zTwKpTd2XuCqf8huc7Fo2iSy+4RHPd10s4KzeTnVr1c=
golang.org/x/net v0.19.0/go.mod h1:CfAk/cbD4CthTvqiEl8NpboMuiuOYsAr/7NOjZJtv1U=
golang.org/x/text v0.14.0 h1:ScX5w1eTa3QqT8oi6+ziP7dTV1S2+ALU0bI+0zXKWiQ=
golang.org/x/text v0.14.0/go.mod h1:18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=
```

## 4. Environment File (`.env`)

```bash
# API Keys (REQUIRED - Replace with your actual keys)
OPENAI_API_KEY=sk-your-openai-api-key-here
ANTHROPIC_API_KEY=sk-ant-your-anthropic-api-key-here
GEMINI_API_KEY=your-gemini-api-key-here

# Cosmos DB (defaults work for emulator)
COSMOS_ENDPOINT=https://localhost:8081
COSMOS_KEY=C2y6yDjf5/R+ob0N8A7Cgv30VRDJIWEHLM+4QDU5DE2nQ9nDuVTqobD4b8mGGyPMbIZnqyMsEcaGQy67XIw/Jw==
COSMOS_DB=deepenc
COSMOS_CONTAINER=messages
COSMOS_PARTITION_KEY=/threadId
COSMOS_EMULATOR_INSECURE=true

# Vector Store
QDRANT_URL=http://localhost:6333
EMBEDDING_MODEL=text-embedding-3-small

# Model Configuration
OPENAI_MODELS=gpt-4o-mini,gpt-4o,gpt-4-turbo-preview
ANTHROPIC_MODELS=claude-3-5-sonnet-20241022,claude-3-opus-20240229
GEMINI_MODELS=gemini-2.0-flash-exp,gemini-1.5-flash,gemini-1.5-pro

# Server
PORT=8080
```

## 5. Setup Script (`setup.sh`)

```bash
#!/bin/bash

set -e

echo "🚀 Deepenc AI - Your private, secure and universal AI"
echo "   Setup - Thread Management with RAG"
echo "============================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker installed${NC}"

# Check Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker Compose installed${NC}"

# Create .env if not exists
if [ ! -f .env ]; then
    echo -e "\n${YELLOW}Creating .env file...${NC}"
    cat > .env << 'EOF'
# API Keys (REQUIRED - Replace with your actual keys)
OPENAI_API_KEY=sk-your-openai-api-key-here
ANTHROPIC_API_KEY=sk-ant-your-anthropic-api-key-here
GEMINI_API_KEY=your-gemini-api-key-here

# Leave other settings as default
EOF
    echo -e "${GREEN}✅ .env created${NC}"
    echo -e "${YELLOW}⚠️  Please edit .env and add your API keys${NC}"
    read -p "Press Enter after adding your API keys..."
fi

# Check API key
source .env
if [[ "$OPENAI_API_KEY" == "sk-your-openai-api-key-here" ]]; then
    echo -e "${RED}❌ Please set your OpenAI API key in .env${NC}"
    exit 1
fi

# Start services
echo -e "\n${YELLOW}Starting services...${NC}"

# Start Cosmos Emulator
echo "Starting Cosmos DB Emulator..."
docker-compose up -d cosmos-emulator
echo "Waiting for Cosmos to initialize..."
sleep 15

# Start Qdrant
echo "Starting Qdrant Vector Store..."
docker-compose up -d qdrant
sleep 5

# Build and start backend
echo "Building backend..."
docker-compose build deepenc-backend

echo "Starting backend..."
docker-compose up -d deepenc-backend
sleep 10

# Check health
echo -e "\n${YELLOW}Checking services...${NC}"

if curl -s http://localhost:8080/api/health > /dev/null; then
    echo -e "${GREEN}✅ Backend is running${NC}"
else
    echo -e "${RED}⚠️  Backend may not be ready${NC}"
fi

if curl -k -s https://localhost:8081/_explorer/index.html > /dev/null; then
    echo -e "${GREEN}✅ Cosmos DB is running${NC}"
else
    echo -e "${RED}⚠️  Cosmos DB may not be ready${NC}"
fi

if curl -s http://localhost:6333/ > /dev/null; then
    echo -e "${GREEN}✅ Qdrant is running${NC}"
else
    echo -e "${RED}⚠️  Qdrant may not be ready${NC}"
fi

echo -e "\n${GREEN}============================================="
echo "✅ Setup Complete!"
echo "=============================================${NC}"
echo ""
echo "Service URLs:"
echo "  Backend API:      http://localhost:8080"
echo "  Health Check:     http://localhost:8080/api/health"
echo "  Cosmos Explorer:  https://localhost:8081/_explorer/index.html"
echo "  Qdrant Dashboard: http://localhost:6333/dashboard"
echo ""
echo "Quick Test:"
echo "  ./test.sh"
echo ""
echo "View Logs:"
echo "  docker-compose logs -f"
echo ""
echo "Stop Services:"
echo "  docker-compose down"
```

## 6. Test Script (`test.sh`)

```bash
#!/bin/bash

echo "🧪 Testing Deepenc AI - Your private, secure and universal AI"
echo "========================"

API_URL="http://localhost:8080"
USER_ID="test-user-123"

# Test health
echo -n "1. Testing health... "
if curl -s "$API_URL/api/health" | grep -q "healthy"; then
    echo "✅"
else
    echo "❌"
    exit 1
fi

# Create thread
echo -n "2. Creating thread... "
THREAD_RESPONSE=$(curl -s -X POST "$API_URL/api/threads" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{"title": "Test Conversation"}')

THREAD_ID=$(echo $THREAD_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ ! -z "$THREAD_ID" ]; then
    echo "✅ Thread: $THREAD_ID"
else
    echo "❌"
    exit 1
fi

# Send message
echo -n "3. Sending message... "
MSG_RESPONSE=$(curl -s -X POST "$API_URL/api/threads/$THREAD_ID/messages" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d '{
    "content": "What is the capital of France?",
    "model": "gpt-4o-mini"
  }')

if echo $MSG_RESPONSE | grep -q "msg_"; then
    echo "✅"
else
    echo "❌"
fi

# Test streaming with context
echo -n "4. Testing contextual streaming... "
echo ""
echo "   Asking: 'Tell me about the capital of France'"
echo "   Response: "
echo ""

curl -X POST "$API_URL/api/stream/contextual" \
  -H "Content-Type: application/json" \
  -d '{
    "threadId": "'$THREAD_ID'",
    "userId": "'$USER_ID'",
    "message": "Tell me more about its famous landmarks",
    "model": "gpt-4o-mini",
    "provider": "openai",
    "context": {
      "strategy": "hybrid",
      "maxMessages": 10,
      "ragEnabled": true,
      "ragCount": 5
    }
  }'

echo ""
echo ""
echo "Test complete! Thread ID: $THREAD_ID"
```

## 7. API Test Examples (`api-examples.md`)

```markdown
# API Examples

## Create Thread
```bash
curl -X POST http://localhost:8080/api/threads \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user123" \
  -d '{"title": "Discussion about AI"}'
```

## List Threads
```bash
curl http://localhost:8080/api/threads \
  -H "X-User-ID: user123"
```

## Send Message
```bash
curl -X POST http://localhost:8080/api/threads/THREAD_ID/messages \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user123" \
  -d '{
    "content": "What is machine learning?",
    "model": "gpt-4o-mini"
  }'
```

## Get Messages
```bash
curl http://localhost:8080/api/threads/THREAD_ID/messages \
  -H "X-User-ID: user123"
```

## Get Context
```bash
curl "http://localhost:8080/api/threads/THREAD_ID/context?query=machine+learning&strategy=hybrid" \
  -H "X-User-ID: user123"
```

## Stream with Context
```bash
curl -X POST http://localhost:8080/api/stream/contextual \
  -H "Content-Type: application/json" \
  -d '{
    "threadId": "THREAD_ID",
    "userId": "user123",
    "message": "Explain neural networks",
    "model": "gpt-4o-mini",
    "provider": "openai",
    "context": {
      "strategy": "hybrid",
      "maxMessages": 10,
      "ragEnabled": true,
      "ragCount": 5
    }
  }'
```
```

## 8. Makefile

```makefile
.PHONY: help setup start stop clean test logs

help:
	@echo "Commands:"
	@echo "  make setup  - Setup everything"
	@echo "  make start  - Start services"
	@echo "  make stop   - Stop services"
	@echo "  make clean  - Remove everything"
	@echo "  make test   - Run tests"
	@echo "  make logs   - View logs"

setup:
	@chmod +x setup.sh test.sh
	@./setup.sh

start:
	@docker-compose up -d
	@echo "Services starting..."
	@sleep 10
	@docker-compose ps

stop:
	@docker-compose down
	@echo "Services stopped"

clean:
	@docker-compose down -v
	@echo "All data removed"

test:
	@chmod +x test.sh
	@./test.sh

logs:
	@docker-compose logs -f

restart: stop start

status:
	@docker-compose ps
	@echo ""
	@curl -s http://localhost:8080/api/health | jq '.' || echo "Backend not responding"
```

## 9. `.gitignore`

```gitignore
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work

# Environment
.env
.env.local

# Certificates
*.crt
*.pem
*.key

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Docker
cosmos-data/
qdrant-data/

# Logs
*.log
```

## Quick Start Instructions

1. **Clone repository and setup:**
```bash
# Make scripts executable
chmod +x setup.sh test.sh

# Run setup
./setup.sh
```

2. **Edit .env and add your API keys**

3. **Start services:**
```bash
make start
# or
docker-compose up -d
```

4. **Test the system:**
```bash
./test.sh
```

5. **Use the API:**
- Create threads for conversations
- Send messages with automatic context
- Use RAG for intelligent retrieval
- Stream responses with context

## Important Notes

- The system works without Redis (no caching)
- RAG features require OpenAI API key for embeddings
- Thread context is automatically managed
- Messages are persisted in Cosmos DB
- Vectors are stored in Qdrant for similarity search