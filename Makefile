run:
	time go run main.go
	ls -la out*
	ls -la testdata/*

test:
	go test ./...

test-logs:
	go test ./... -v -count 1

clean: clean-cache clean-out
clean-cache:
	rm -rf ./cache ./encoder/cache
clean-out:
	rm -f ./out*
