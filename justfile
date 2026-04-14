default:
    @just --list

dev:
    wgo -file=.go -file=.env go run cmd/walens/main.go \
        :: wgo -file=.env -cd frontend npm run dev
