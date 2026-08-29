.PHONY: provenance

provenance:
	go mod verify
	test -z "$$(go list -m -f '{{if .Replace}}{{.Path}}{{end}}' all)"
	test "$$(openssl dgst -sha256 typeid/testdata/official/valid.json | awk '{print $$NF}')" = "af5a9cf2447d757b9354f33861d5d83f4e3244487eac191e37021013ce0c17e3"
	test "$$(openssl dgst -sha256 typeid/testdata/official/invalid.json | awk '{print $$NF}')" = "b1bf19bd2c922970bbe0381499807d3a0da652abb3da813b9ab75bac7c25217c"
	grep -F '// Source-map SHA-256: e089edb1b32df867bf27f3bca224228f49a850a75bfa4f693d293306e83db72c' slug/transliteration_table.go >/dev/null
	test "$$(openssl dgst -sha256 slug/testdata/laravel_english.json | awk '{print $$NF}')" = "56cc72e173eed7efb6b6ae6f738bcdde075c4f0e66e54190e89fdff40fbc88e9"
	awk -F '\t' -f verification/vector-provenance.awk specification/vector-provenance.tsv
