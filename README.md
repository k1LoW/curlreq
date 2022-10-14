# curlreq

`curlreq` creates `*http.Request` from [curl](https://curl.se/) command.

## Usage

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/k1LoW/curlreq"
)

func main() {
	cmd := "curl https://example.com"
	req, err := curlreq.NewRequest(cmd)
	if err != nil {
		log.Fatal(err)
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)

	// Output:
	// 200
}
```

## Reference

- [tj/parse-curl.js](https://github.com/tj/parse-curl.js): Parse curl commands, returning an object representing the request.
