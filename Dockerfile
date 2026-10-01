# -----------------------------------------------------------------------------------------------
#                             Stage 1: Builder 
# -----------------------------------------------------------------------------------------------

# Builder has all the required elements for selfexecuting (code, golang, dependencies, exra packages etc.)

FROM golang:1.26-alpine AS Builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

RUN apk add --no-cache build-base #Το apk είναι ο διαχειριστής πακέτων του Alpine. Το build-base εγκαθιστά τον C compiler και τα βασικά εργαλεία compilation, ενώ το --no-cache αποφεύγει την αποθήκευση του cache των πακέτων.

COPY . .

RUN go build -o forum ./cmd/main.go


# -----------------------------------------------------------------------------------------------
#                             Stage 2: Create minimal image
# -----------------------------------------------------------------------------------------------

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/forum .
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["./forum"]


