test_static:
	gometalinter --enable-all --min-confidence=0.3 --line-length=120 \
		-e "parameter \w+ always receives" \
		-e "/jinzhu/gorm/" \
		-e "model is unused" \
		-e '"expections" is a misspelling of "exceptions"' \
		./...

test_unit:
	 go test -v ./...

test: test_unit bench test_static

bench:
	go test -bench=. -benchtime=5s -v -run=^$$ ./...
