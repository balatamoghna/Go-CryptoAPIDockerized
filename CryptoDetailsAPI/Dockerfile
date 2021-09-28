FROM golang:alpine as builder
# ENV GO111MODULE=on
RUN apk update && apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download 
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-api .



# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage. Observe we also copied the .env file
COPY --from=builder /app/go-api .      
 

# Expose port 3000 to the outside world
EXPOSE 3000


#Command to run the executable
CMD ["./go-api"]