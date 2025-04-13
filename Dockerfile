 # Base image
FROM golang:1.24

# Set working directory
WORKDIR /app

# Copy go files
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

# Build the Go binary
RUN go build -o main .

# Run the app
CMD ["/app/main"]

