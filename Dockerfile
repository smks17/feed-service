FROM golang:alpine

RUN mkdir /app
WORKDIR /app

# build
RUN apk update && apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main ./api/

# deploy
RUN ls -ltrh
ENV PORT=8080
EXPOSE 8080
RUN chmod +x /app/main
CMD ["/app/main"]
