It's dangerous to go-alone! Take this.
==========================================
[![GoDoc](https://godoc.org/github.com/bwmarrin/go-alone?status.svg)](https://godoc.org/github.com/bwmarrin/go-alone) [![Go report](http://goreportcard.com/badge/bwmarrin/go-alone)](http://goreportcard.com/report/bwmarrin/go-alone) [![Build Status](https://travis-ci.org/bwmarrin/go-alone.svg?branch=master)](https://travis-ci.org/bwmarrin/go-alone) [![Coverage](http://gocover.io/_badge/github.com/bwmarrin/go-alone)](https://gocover.io/github.com/bwmarrin/go-alone) [![Discord Gophers](https://img.shields.io/badge/Discord%20Gophers-%23info-blue.svg)](https://discord.gg/0f1SbxBZjYq9jLBk)

<img align="right" src="https://raw.githubusercontent.com/wiki/bwmarrin/go-alone/8bitsword.png">

go-alone is a [Go](https://golang.org/) package that provides
* Methods to create and verify [MAC](https://en.wikipedia.org/wiki/Message_authentication_code) signatures of data
* Ability to add timestamps to signed tokens and use custom epoch if needed.
* BLAKE2b signatures and Base58 time encoding provides outstanding performance and security.
* A very simple to use API with good documentation and 100% test coverage.
* Various helper methods for parsing tokens

**For more information, please read the [wiki](https://github.com/bwmarrin/go-alone/wiki)**

**For help with this package or general Go discussion, please join the [Discord 
Gophers](https://discord.gg/0f1SbxBZjYq9jLBk) chat server.**

**For a fast and easy to use snowflake ID library, check out [this](https://github.com/bwmarrin/snowflake)**

## Getting Started
This assumes you already have a working Go environment, if not please see
[this page](https://golang.org/doc/install) first.

### Installing


```sh
go get github.com/bwmarrin/go-alone
```

### Usage

Here's a basic example below.  There is also an example program in the [example](https://github.com/bwmarrin/go-alone/tree/master/example)
folder that demonstrates a few more ways of using Go-Alone. You can read the API
documentation on [GoDoc](https://godoc.org/github.com/bwmarrin/go-alone).

```go
package main

import (
	"github.com/bwmarrin/go-alone"
)

func main() {

	// This secret is used as the hash key for the signer.
	var secret = []byte("It's a secret to everybody")

	// This data is what we will be signing below.
	var data = []byte("It's dangerous to go alone! Take this.")

	// Create a new Signer using our secret
	s := goalone.New(secret)

	// Sign and return a token in the form of `data.signature`
	token := s.Sign(data)

	// Unsign the token to verify it - if successful the data portion of the
	// token is returned.  If unsuccessful then d will be nil, and an error
	// is returned.
	d, err := s.Unsign(token)
	if err != nil {
		// signature is not valid. Token was tampered with, forged, or maybe it's
		// not even a token at all! Either way, it's not safe to use it.
	} else {
		// signature is valid, it is safe to use the data
		println(string(d))
	}
}
```


### Performance / Testing

To run the tests and benchmarks, use the following command.

```sh
go test -bench=. -v
```
