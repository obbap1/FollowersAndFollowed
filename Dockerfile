FROM golang:1.16.3-alpine

WORKDIR /app
# Copy and download avalanche dependencies using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

RUN go build -o /executable

CMD [ "/executable" ]


