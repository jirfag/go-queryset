test_static:
	gometalinter --vendor --enable-all --min-confidence=0.3 --line-length=120 \
		-e "parameter \w+ always receives" \
		-e "/jinzhu/gorm/" \
		-e "model is unused" \
		-e '"expections" is a misspelling of "exceptions"' \
		./...

test_unit:
	mkdir -p test
	go test -v ./parser/ ./queryset/

test: test_unit bench test_static

bench:
	go test -bench=. -benchtime=1s -v -run=^$$ ./queryset/

gen:
	 go generate ./queryset/test/
