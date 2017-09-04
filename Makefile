test_static:
	gometalinter --enable-all --min-confidence=0.3 --line-length=120 \
		-e "parameter \w+ always receives" \
		-e "/jinzhu/gorm/" \
		-e "field model is unused" \
		./...

test_unit:
	 go test -v ./...

test: test_unit test_static
