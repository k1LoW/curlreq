package curlreq_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/curlreq"
)

// TestParse ref: https://github.com/tj/parse-curl.js
func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  *curlreq.Parsed
	}{
		{
			`curl http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{},
			},
		},
		{
			`curl -I http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodHead,
				Header: http.Header{},
			},
		},
		{
			`curl -I http://api.sloths.com -vvv --foo --whatever bar`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodHead,
				Header: http.Header{},
			},
		},
		{
			`curl -H "Origin: https://example.com" https://example.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://example.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Origin": []string{"https://example.com"},
				},
			},
		},
		{
			`curl --compressed http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept-Encoding": []string{"deflate, gzip"},
				},
			},
		},
		{
			`curl -H "Accept-Encoding: gzip" --compressed http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept-Encoding": []string{"gzip"},
				},
			},
		},
		{
			`curl -X DELETE http://api.sloths.com/sloth/4`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com/sloth/4"),
				Method: http.MethodDelete,
				Header: http.Header{},
			},
		},
		{
			`curl -XPUT http://api.sloths.com/sloth/4`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com/sloth/4"),
				Method: http.MethodPut,
				Header: http.Header{},
			},
		},
		{
			`curl https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{},
			},
		},
		{
			`curl -u tobi:ferret https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Authorization": []string{"Basic dG9iaTpmZXJyZXQ="},
				},
			},
		},
		{
			`curl -d "foo=bar" https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   "foo=bar",
			},
		},
		{
			`curl -d "foo=bar" -d bar=baz https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   "foo=bar&bar=baz",
			},
		},
		{
			`curl -H "Accept: text/plain" --header "User-Agent: slothy" https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/plain"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
		{
			`curl -H 'Accept: text/*' --header 'User-Agent: slothy' https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/*"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
		{
			`curl -H 'Accept: text/*' -A slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/*"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
		{
			`curl -b 'foo=bar' slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Cookie": []string{"foo=bar"},
				},
			},
		},
		{
			`curl --cookie 'foo=bar' slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Cookie": []string{"foo=bar"},
				},
			},
		},
		{
			`curl --cookie 'species=sloth;type=galactic' slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Cookie": []string{"species=sloth;type=galactic"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := curlreq.Parse(tt.input)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Errorf("%s", diff)
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			`curl http://example.com`,
			`{"url":"http://example.com","method":"GET","header":{}}`,
		},
		{
			`curl 'http://google.com/' \
  -H 'Accept-Encoding: gzip, deflate, sdch' \
  -H 'Accept-Language: en-US,en;q=0.8,da;q=0.6' \
  -H 'Upgrade-Insecure-Requests: 1' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.110 Safari/537.36' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' \
  -H 'Connection: keep-alive' \
  --compressed`,
			`{"url":"http://google.com/","method":"GET","header":{"Accept":["text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"],"Accept-Encoding":["gzip, deflate, sdch"],"Accept-Language":["en-US,en;q=0.8,da;q=0.6"],"Connection":["keep-alive"],"Upgrade-Insecure-Requests":["1"],"User-Agent":["Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.110 Safari/537.36"]}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := curlreq.Parse(tt.input)
			if err != nil {
				t.Error(err)
			}
			got, err := json.Marshal(p)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(string(got), tt.want, nil); diff != "" {
				t.Errorf("%s", diff)
			}
		})
	}
}

func Example() {
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

func URL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
