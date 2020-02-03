default: all
.PHONY: clean all default
# TODO: version lib
all: datamon.so
# Define this in build environment (2.7 is default)
PYTHON_MAJOR_VERSION ?= 2.7
clean:
	@rm -rf _obj *.so *.pyc datamon.h datamon_wrapper.go # go.mod go.sum
authorizer.h: ../cgo/authorizer_wrapper.go
	#cp ../cgo/go.mod ../cgo/go.sum . ;
	GO111MODULE=on go tool cgo -exportheader $@  -- `pkg-config --cflags-only-I python-${PYTHON_MAJOR_VERSION}` $< ; \
	cp $< .
datamon.so: ../cgo/datamon_wrapper.go datamon_wrapper.c datamon.h
	CGO_CPPFLAGS=`pkg-config --cflags-only-I python-${PYTHON_MAJOR_VERSION}` ; export CGO_CPPFLAGS ; \
	CGO_CFLAGS=`pkg-config --cflags-only-I python-${PYTHON_MAJOR_VERSION}` ; export CGO_CFLAGS ; \
	CGO_LDFLAGS=`pkg-config --libs python-${PYTHON_MAJOR_VERSION}` ; export CGO_LDFLAGS ; \
	GO111MODULE=on go build -buildmode=c-shared -o $@ ; \
	chmod ugo+x $@
