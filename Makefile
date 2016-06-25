serve: www
	go run ./server/*

test: go-test

go-test:

clean:
	rm -rf www client.js client.js.map bundle.js bundle.js.map pre-bundle.js

node_modules:
	npm install

bundle.js: client.js
	browserify --debug . -o pre-bundle.js
#	cat pre-bundle.js | exorcist bundle.js.map > bundle.js
	cp pre-bundle.js $@

www/client.js: bundle.js
	mkdir -p www
	cp bundle.js $@
#	uglifyjs bundle.js -c -m -o $@
# 	uglifyjs bundle.js -c -m -o $@ \
# 		--source-map @$.map \
# 		--source-map-root /js \
# 		--source-map-url /js/client.js.map \
# 		--in-source-map bundle.js.map

www: www/client.js
	mkdir -p www
	cp -a html/* www
	cp -a vendor/*/* www
	cp -a slides www/slides

client.js: node_modules client/main.go client/main.inc.js
	gopherjs build client/* -o client.js

all: clean serve
